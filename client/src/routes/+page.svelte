<script lang="ts">
  import { onMount } from 'svelte';
  import {EditorState} from "@codemirror/state"
  import {EditorView, keymap} from "@codemirror/view"
  import {defaultKeymap} from "@codemirror/commands"

  onMount(() => {
    let startState = EditorState.create({
    doc: "Hello World",
    extensions: [keymap.of(defaultKeymap)]
  })

  let view = new EditorView({
    state: startState,
    parent: document.getElementById("codemirror-editor") ?? document.body
  });
  })
</script>

<svelte:head>
	<link href="//cdn.quilljs.com/1.3.6/quill.snow.css" rel="stylesheet">
</svelte:head>
<main>
  <h1>Welcome to MantaDocs</h1>
  <!-- <div bind:this={editorContainer}></div> -->
   <textarea id="editor"></textarea>
   <div id="codemirror-editor"></div>
</main>

<style lang="scss">

</style>
