import type { Identifier, Char } from "$lib/types/CRTD-types";

// Initialize WebSocket
const socket: WebSocket = new WebSocket("ws://localhost:8080/ws");

/**
 * Function to send a single character change to server and all other clients
 * 
 * An character index is represented as such:
 * INDEX = {
 *  digit: int,
 *  siteId: string,
 *  lamport: int
 * }
 * 
 * The general JSON format of a character change should be as follows:
 * SingleChange = {
 *  changeType: "single",
 *  position: INDEX[],
 *  char: string,
 *  changeEvent: string,
 *  lamport: number
 * }
 */

enum ChangeEvent {
  insert = "insert",
  delete = "delete"
}

enum ChangeType {
  single = "single",
  batch = "batch"
}

type SingleChange = {
  position: Identifier[],
  char: string,
  lamport: number,
  changeEvent: ChangeEvent
}

type SingleChangeMessage = { changeType: ChangeType.single } & SingleChange;

type BatchChange = {
  changeType: ChangeType.batch,
  changes: SingleChange[]
}

function sendSingleChange(c: Char, event: ChangeEvent) {
  if (socket && socket.readyState === WebSocket.OPEN) {
    const newSingleChange: SingleChangeMessage = {
      changeType: ChangeType.single,
      position: c.position,
      char: c.value,
      lamport: c.lamport,
      changeEvent: event
    }
    socket.send(JSON.stringify(newSingleChange));
    console.log("Message sent!");
  } else {
    console.error('WebSocket connection is not open');
  }
}

socket.onopen = () => {
  console.log('WebSocket connection opened!');
};

socket.onmessage = (event) => {
  console.log("Message received:", event.data);
};

export default {sendSingleChange}
export {ChangeEvent, ChangeType}