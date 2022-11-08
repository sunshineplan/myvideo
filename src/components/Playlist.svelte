<script lang="ts">
  import { onMount } from "svelte";

  export let loading: number;
  export let name = "";
  export let url: string;
  export let playlist: { [key: string]: play[] } = {};

  let current = "";

  const open = async (title: string, play: play) => {
    if (play.ep == "暂无资源") return;
    loading++;
    const resp = await fetch("/play", {
      method: "post",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ url, play: play.m3u8 }),
    });
    loading--;
    if (resp.ok) {
      const url = await resp.text();
      window.open(`/play?title=${title} - ${play.ep}&url=${url}`);
      return;
    }
    alert("Failed to get play");
  };

  onMount(() => {
    if (Object.keys(playlist).length) current = Object.keys(playlist)[0];
  });
</script>

<ul class="nav nav-tabs">
  {#each Object.keys(playlist) as src}
    <li class="nav-item">
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <span
        class="nav-link"
        class:active={current == src}
        on:click={() => (current = src)}
      >
        {src}
      </span>
    </li>
  {/each}
</ul>
<div class="items">
  {#if playlist[current]}
    {#each playlist[current] as play (play.ep)}
      <li>
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <span class="play" on:click={() => open(name, play)}>
          {play.ep}
        </span>
      </li>
    {/each}
  {/if}
</div>

<style>
  .nav {
    margin-bottom: 10px;
    font-size: 14px;
  }

  .nav-link {
    cursor: default;
    padding: 0.5rem;
  }

  .items {
    height: calc(100% - 52px);
    overflow-y: auto;
  }

  .items > li {
    display: inline-block;
    margin: 10px 6px;
    cursor: pointer;
  }

  .play {
    border: 1px solid #6c757d;
    border-radius: 3px;
    padding: 5px;
    color: #343a40;
  }

  ::-webkit-scrollbar-thumb {
    background: #6c757d;
  }

  ::-webkit-scrollbar {
    width: 3px;
  }
</style>
