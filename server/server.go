package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"os"
	"context"
	"time"

	"github.com/joho/godotenv"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CharIdentifier struct {
	Digit	int 	`json:"digit" bson:"digit"`
	SiteID	string	`json:"siteID" bson:"siteID"`
}

// Key for hashmap which will store chars
type Position []CharIdentifier;

type Char struct {
	Position	Position	`json:"position" bson:"position"`
	Lamport		int			`json:"lamport" bson:"-"`
	Value		string		`json:"value" bson:"value"`
}

type LineDO struct {
	Chars	[]Char	`json:"chars" bson:"chars"`
}

type DocumentDO struct {
	DocID	primitive.ObjectID	`json:"docID" bson:"_id,omitempty"`
	Lines	[]LineDO			`json:"lines" bson:"lines"`
}

type CharHashMap map[string]string


type ActiveDocs struct {
	mu				sync.Mutex
	Docs 			map[string]*Document	// Map of all active documents
	MongoDBClient	*mongo.Client			// MongoDB Client object
}

type Document struct {
	mu sync.Mutex
	DocID string			// Document ID
	CRTD *CharHashMap		// CRTD hashmap; holds position-char hashmap
	ClientRepresent *[]Char
	Users map[string]*User	// List of current users/editors
}

type User struct {
	UserID string
	Socket *websocket.Conn	// Websocket Connection
	Doc *Document			// Document the user is connected to
}



// Char or Cursor event
type ChangeEvent string
const (
	ChangeInsert ChangeEvent = "insertChar"
	ChangeDelete ChangeEvent = "deleteChar"
	ChangeCursor ChangeEvent = "changeCursor"
	ChangeSelect ChangeEvent = "selectText"
)

type ChangeType string
const (
	ChangeTypeSingle = "singleCharChange"
	ChangeTypeBatch = "batchCharChange"
	ChangeTypeCursor = "cursorChange"
)

type SingleCharChange struct {
	Position	Position		`json:"position"`
	Char		string			`json:"char"`
	ChangeEvent	ChangeEvent		`json:"changeEvent"`
	Lamport		int				`json:"lamport"`
}

type BatchCharChange struct {
	Changes []SingleCharChange	`json:"changes"`
}

type CursorChange struct {
	NewPosition Position	`json:"newPosition"`
}

type SelectChange struct {
	FromPosition Position	`json:"fromPosition"`
	Length int				`json:"length"`
}

type MessageInfo struct {
	Type ChangeType	`json:"changeType"`
	UserID string	`json:"userID"`
}

// Validate that SingleCharChange has an allowed value
func (s *SingleCharChange) Validate() error {
	switch s.ChangeEvent {
	case ChangeInsert, ChangeDelete:
		return nil
	default:
		return errors.New("invalid change type")
	}
}

// Validate BatchCharChange
func (b *BatchCharChange) Validate() error {
	for _, change := range b.Changes {
		if err := change.Validate(); err != nil {
			return err
		}
	}
	return nil
}


var allowedOrigins = [...]string{
	"http://localhost:5173", // Localhost testing
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Resolve cross-domain problems
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("origin")
		log.Println("ORIGIN GIVEN:", origin);
		for _, allowOrigin := range allowedOrigins {
			if origin == allowOrigin {
				return true
			}
		}
		return false
	},
}

func makeHashMapKey(position Position, lamport int) string {
	var strBuilder strings.Builder
	for _, ident := range(position) {
		strBuilder.WriteString("D.")
		strBuilder.WriteString(strconv.Itoa(ident.Digit))
		strBuilder.WriteString(".S.")
		strBuilder.WriteString(ident.SiteID)
	}
	strBuilder.WriteString(".L.")
	strBuilder.WriteString(strconv.Itoa(lamport))
	return strBuilder.String()
}

func makePositionAndLamport(hashKey string) (Position, int, error) {
	hashStrings := strings.Split(hashKey, ".");
	var position Position
	var currDigit int
	var currSiteID string
	var lamportVal int
	var err error
	for i := 0; i < len(hashStrings); i+=2 {
		marker := hashStrings[i]
		switch(marker) {
		case "D":
			digit, parseErr := strconv.Atoi(hashStrings[i + 1])
			if (parseErr != nil) {
				err = parseErr
				break
			}
			currDigit = digit
		case "S":
			currSiteID = hashStrings[i + 1]
			newIdentifier := CharIdentifier{
				Digit: currDigit,
				SiteID: currSiteID,
			}
			position = append(position, newIdentifier)
		case "L":
			lamport, parseErr := strconv.Atoi(hashStrings[i + 1])
			if (parseErr != nil) {
				err = parseErr
				break
			}
			lamportVal = lamport
		}
	}
	return position, lamportVal, err
}



// ----------------------------- Main --------------------------- //
func main() {
	log.Println("Hello, World! Initializing MantaDocs server...")

	// Attempt to load .env file for local development
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, assuming production environment")
	}

	// Loading environment variables
	dbURITemplate := os.Getenv("MONGO_DB_URI_TEMPLATE")
	dbUsername := os.Getenv("MONGO_DB_USERNAME")
	dbPassword := os.Getenv("MONGO_DB_PASSWORD")
	if dbURITemplate == "" || dbUsername == "" || dbPassword == "" {
		log.Fatalf("Environment variables MONGO_DB_URI_TEMPLATE, MONGO_DB_USERNAME, and MONGO_DB_PASSWORD must be set")
	}

	// Set MongoDB URI and auth info
	MONGO_URI := strings.Replace(dbURITemplate, "{username}", dbUsername, 1)
	MONGO_URI = strings.Replace(MONGO_URI, "{password}", dbPassword, 1)

	// Setting parameters and context for MongoDB (Atlas)
	serverAPIOpts := options.ServerAPI(options.ServerAPIVersion1)
	clientOpts := options.Client().ApplyURI(MONGO_URI).SetServerAPIOptions(serverAPIOpts)
	ctx, ctxCancelFunc := context.WithTimeout(context.Background(), 10*time.Second) // Connection times out if delay >10sec

	// Connect to database
	mongoDBClient, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		panic(err)
	}
	defer mongoDBClient.Disconnect(ctx) // When main function closes, disconnect client
	defer ctxCancelFunc() // Cancel context after main function closes

	// Ping database
	if err = mongoDBClient.Ping(ctx, readpref.Primary()); err != nil {
		panic(err)
	}
	log.Println("Successfully pinged MongoDB database!")

	// Get 'documents' collection
	mantaDocDB := mongoDBClient.Database("mantadocs")
	documentCollection := mantaDocDB.Collection("documents")

	// Test finding document based on docID
	id, err := primitive.ObjectIDFromHex("66bff0b1e218c010175acf6a")
	if err != nil {
		log.Fatal(err)
	}
	var testDoc DocumentDO
	filter := bson.M{"_id": id}
	err = documentCollection.FindOne(ctx, filter).Decode(&testDoc)
	if err != nil {
        log.Fatal(err)
    }
	log.Printf("Found Doc: %+v\n", testDoc)

	// Create server state
	docMap := make(map[string]*Document)
	activeDocs := &ActiveDocs{Docs: docMap, MongoDBClient: mongoDBClient}

	// HTTP Handlers
	log.Println("Making routes...")
	http.HandleFunc("GET /doc/{docID}", activeDocs.getDocHandler)
	http.HandleFunc("/ws", activeDocs.webSocketHandler)
	http.ListenAndServe(":8080", nil)
}



func (AD *ActiveDocs) getDocHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	docIDHex := r.PathValue("docID")
	var retrieveDocJSON []byte

	// Thread safety
	AD.mu.Lock()
	defer AD.mu.Unlock()

	_, ok := AD.Docs[docIDHex]
	if !ok {
		log.Printf("Retrieving document w/ ID '%s'...\n", docIDHex)
		documentCollection := AD.
			MongoDBClient.
			Database("mantadocs").
			Collection("documents")

		docID, err := primitive.ObjectIDFromHex(docIDHex)
		if err != nil {
			log.Panic(err)
		}

		var retrieveDoc DocumentDO
		filter := bson.M{"_id": docID}
		err = documentCollection.FindOne(ctx, filter).Decode(&retrieveDoc)
		if err != nil {
			log.Panic(err)
			if err == mongo.ErrNoDocuments {
				// Handle the case where no document was found
				http.Error(w, "No document found", http.StatusNotFound)
				return
			} else if ctx.Err() != nil {
				// Check if the context was done (timeout or cancellation)
				if ctx.Err() == context.DeadlineExceeded {
					http.Error(w, "Request timed out", http.StatusRequestTimeout)
				} else if ctx.Err() == context.Canceled {
					http.Error(w, "Request canceled", http.StatusRequestTimeout)
				}
				return
			}
			// If any other error occurs
			http.Error(w, "Failed to query database", http.StatusInternalServerError)
			return
		}

		// Marshal the DocumentDO to JSON
		retrieveDocJSON, err = json.Marshal(retrieveDoc)
		if err != nil {
			http.Error(w, "Failed to marshal MongoDB doc data to JSON", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(retrieveDocJSON)
	} else {
		// Set header
		w.Header().Set("Content-Type", "application/json")

	}
}

func (AD *ActiveDocs) webSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil) // Upgrade http to websocket protocol (http uses TCP)
    if err != nil {
        log.Println(err)
        return
    }

	fmt.Println("Connected to web socket!")

	for {
		// Read a message from the client.
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}

		// Print the message to the console.
		fmt.Println("Message received!")
		fmt.Println("Message type:", messageType)
		processJSONMessage(message);


		// // Send a message back to the client.
		// err = conn.WriteMessage(messageType, []byte("Hello, client!"))
		// if err != nil {
		// 	fmt.Println(err)
		// 	return
		// }
	}
}

func processJSONMessage(messageBytes []byte) error {
	var msgInfo MessageInfo
	err := json.Unmarshal(messageBytes, &msgInfo)
	if err != nil {
		return fmt.Errorf("MessageInfo unmarshall ERROR: %s", err)
	}

	switch msgInfo.Type {
	case ChangeTypeSingle:
		var singleChange SingleCharChange
		err = json.Unmarshal(messageBytes, &singleChange)
		fmt.Println("SingleCharChange detected!")
		fmt.Printf("%+v\n", singleChange)

	case ChangeTypeBatch:
		var batchChange BatchCharChange
		err = json.Unmarshal(messageBytes, &batchChange)
		fmt.Println("Batch Change detected!")
		fmt.Printf("%+v\n", batchChange)
	
	case ChangeTypeCursor:
		var cursorChange CursorChange
		err = json.Unmarshal(messageBytes, &cursorChange)
		fmt.Println("Cursor Change detected!")
		fmt.Printf("%+v\n", cursorChange)

	default:
		return fmt.Errorf("ERROR: unknown change type")
	}

	if err != nil {
		return fmt.Errorf("unmarshall ERROR: %s", err)
	}
	return nil
}

func performCharOperation(hashKey string, charChange SingleCharChange) error {
	// switch(charChange.ChangeEvent){
	// case ChangeInsert:
	// 	if _, ok := charHashMap[hashKey]; ok {
	// 		return nil
	// 	}
	// 	charHashMap[hashKey] = charChange.Char
	// case ChangeDelete:
	// 	if _, ok := charHashMap[hashKey]; !ok {
	// 		return nil
	// 	}
	// 	delete(charHashMap, hashKey)
	// default:
	// 	return fmt.Errorf("unknown change event: %s", charChange.ChangeEvent)
	// }
	// return nil
	return nil
}