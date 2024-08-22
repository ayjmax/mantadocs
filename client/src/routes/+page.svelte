<script lang="ts">
  import type { CRTD_Representation, Identifier, Char } from '$lib/types/CRTD-types';
  import { generatePositionBetween } from '$lib/functions/CRDT';
  import LamportClock from '$lib/functions/lamport-clock';
  import { v4 as uuidv4 } from 'uuid';
  import { onMount } from 'svelte';
  import Quill, { Range } from 'quill';
  import QuillCursors from 'quill-cursors';
  import ws, {ChangeEvent} from '$lib/functions/websocket';

  Quill.register('modules/cursors', QuillCursors);
  let editorContainer: HTMLDivElement;
  let quill: Quill;
  let content: any;
  let CRD = null;
  let lamportClock: LamportClock = new LamportClock();
  let prevCursorIndex: number = 0;

  function sendCharToServer() {
    ws.sendSingleChange({
      position: [
        {digit: 1, siteID: "test_siteId"}
      ],
      lamport: 1,
      value: "%"
    }, ChangeEvent.insert);
  }

  function charsToString(chars: Char[]) {
    return chars.map(c => c.value).toString();
  }

  onMount(() => {
    // Testing CRDT
    const siteID = uuidv4();
    let chars: Char[] = [];

    // Create Quill Editor
    quill = new Quill(editorContainer, {
      theme: 'snow',
      modules: {
        toolbar: [
          [{ header: [1, 2, false] }],
          ['bold', 'italic', 'underline'],
          ['image', 'code-block']
        ],
      }
    });

    // Handle text change
    quill.on('text-change', (delta, oldDelta, source) => {
      setTimeout(() => {
        if (source !== 'user') {return;}

        console.log("Delta:", delta);
        let startDeltaCursorIndex = 0;
        // let currCursorIndex = quill.getSelection()?.index;
        // if (currCursorIndex === undefined) {return;}

        for (const op of delta.ops) {
          if (op.retain && typeof op.retain === 'number') {
            startDeltaCursorIndex = op.retain;
          }
          if (op.insert && typeof op.insert === 'string') {
            // let behindChar = chars.at(startDeltaCursorIndex - 2);
            let behindChar = chars.at(startDeltaCursorIndex - 1);
            const infrontChar = chars.at(startDeltaCursorIndex);
            const insertedString = op.insert;

            console.groupCollapsed('Insert Operation Info');
              console.log('op:', op);
              console.log('behindChar: "%s"', behindChar?.value);
              console.log('infrontChar: "%s"', infrontChar?.value);
              console.log('startDeltaCursorIndex:', startDeltaCursorIndex);

            for (let i = 0; i < insertedString.length; i++, startDeltaCursorIndex++) {
              const newIndenList = generatePositionBetween(
                (behindChar?.position || []), (infrontChar?.position || []), siteID
              );
              const newChar: Char = {
                position: newIndenList,
                lamport: lamportClock.tick(),
                value: insertedString.charAt(i)
              };
              // chars.splice(startDeltaCursorIndex - 1, 0, newChar);
              chars.splice(startDeltaCursorIndex, 0, newChar);
              behindChar = newChar;

              console.group(`Inserting: "${insertedString.charAt(i)}"`);
                console.log("startDeltaCursorIndex:", startDeltaCursorIndex);
                console.log("CHARS:", charsToString(chars));
              console.groupEnd();
            }
            console.groupEnd();
          } else if (op.delete) {
            const deletedChars: Char[] = chars.splice(startDeltaCursorIndex, op.delete);
            console.groupCollapsed('Delete Operation Info');
              console.log('op:', op);
              console.log('startDeltaCursorIndex:', startDeltaCursorIndex);
              console.log("Deleted chars:", charsToString(deletedChars));
              console.log("CHARS:", charsToString(chars));
            console.groupEnd();
          }
        }
      }, 1); // Have to delay by 1 ms due to inconsistent cursor index positioning on first char
    });

    // Handle cursor position change
    quill.on('editor-change', (eventName, ...args) => {
      if (eventName !== 'selection-change') {return;}
      if (!(args[0] instanceof Range && args[1] instanceof Range)) {return;}

      console.groupCollapsed("Cursor Info");
        console.log("old cursor:", args[1]);
        console.log("current cursor:", args[0]);
      console.groupEnd();
    });
  });


</script>


<svelte:head>
	<link href="//cdn.quilljs.com/1.3.6/quill.snow.css" rel="stylesheet">
</svelte:head>

<main>
  <h1>Welcome to MantaDocs</h1>
  <button on:click={sendCharToServer}>
    Click to send char to server!
  </button>
  <div id="quill-editor" bind:this={editorContainer}></div>
</main>




<style lang="scss">
  main {
    display: flex;
    min-height: 100vh;
    flex-direction: column;
    justify-content: stretch;
  }
  /* #quill-editor {
    display: flex;
    flex-grow: 1;
    margin: 3rem;
    border: 2px solid grey;
  } */
</style>
