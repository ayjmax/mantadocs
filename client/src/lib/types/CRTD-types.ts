/*
  * A position identifier generated at site s is a tuple (position, clocks) where
      the clock(s) is the Lamport clock value at site s.
  * A position is a list of identifiers.
  * An identifier is a tuple (digit, site id) where digit and site id are integers.
*/

/*
  The digit will be BASE-256 and convert the number to a string for ease of debugging,
  but other reasonable choices include MAX_INT, Base64, Base85, etc.

  Base 256 is great as this fits the 256 bits in a byte,
  so up to 256 can be represented as a single ASCII character.

  NOTE: Each position is between 0 and 1 insofar as the digits are concerned,
  so we can think of a position as p=0.p1p2p3… where pi are identifiers and
  we don’t need to store anything left of the decimal.
*/
export type Identifier = {
  digit: number,  // Base 256 number
  siteID: string    // UUID from server
}

export type Char = {
  position: Identifier[]
  lamport: number
  value: string
}

export const DECIMAL_BASE = 256;

export type CRTD_Representation = Char[];