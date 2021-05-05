<script lang="ts">
  import { onMount } from "svelte";
  import Playlist from "./components/Playlist.svelte";

  const category: Category = {
    dongman: "动漫",
    dianying: "电影",
    dianshiju: "电视剧",
    zongyi: "综艺",
  };
  const keys = Object.keys(category) as Array<keyof Category>;

  let query = "";
  let list: Partial<List<Category>> = {};
  let current: video[] = [];
  let loading = 0;

  const channel = async (category?: keyof Category) => {
    let url: string;
    if (category && category != "dongman") url = "/list?c=" + category;
    else url = "/list";
    if (!category) category = "dongman";
    if (list[category]) {
      current = list[category] as video[];
      return;
    }
    const li = await getList(url);
    if (li) {
      list[category] = li;
      current = li;
    }
  };

  const search = async (query?: string) => {
    let url: string;
    if (query) url = "/list?q=" + query;
    else url = "/list";
    const list = await getList(url);
    if (list) current = list;
  };

  const getList = async (url: string) => {
    loading++;
    const resp = await fetch(url);
    loading--;
    if (resp.ok) {
      const json = await resp.json();
      if (Array.isArray(json)) {
        return json as video[];
      } else if (!json) {
        alert("No video found");
        return;
      }
    }
    alert("Failed to get list");
  };

  onMount(async () => {
    await channel();
  });
</script>

<header class="navbar navbar-expand flex-column flex-md-row">
  <a class="navbar-brand text-primary" href="/">My Video</a>
  <ul class="navbar-nav">
    {#each keys as c}
      <li class="nav-link" on:click={async () => await channel(c)}>
        {category[c]}
      </li>
    {/each}
  </ul>
  <div class="input-group">
    <input
      class="form-control"
      type="search"
      bind:value={query}
      placeholder="Search Video"
      on:keydown={async (e) => {
        if (e.key === "Escape") query = "";
        else if (e.key === "Enter") await search(query);
      }}
    />
    <button
      class="btn btn-outline-primary"
      on:click={async () => await search(query)}
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="16"
        height="16"
        fill="currentColor"
        viewBox="0 0 16 16"
      >
        <path
          d="M11.742 10.344a6.5 6.5 0 1 0-1.397 1.398h-.001c.03.04.062.078.098.115l3.85 3.85a1 1 0 0 0 1.415-1.414l-3.85-3.85a1.007 1.007 0 0 0-.115-.1zM12 6.5a5.5 5.5 0 1 1-11 0 5.5 5.5 0 0 1 11 0z"
        />
      </svg>
    </button>
  </div>
</header>
<div class="content" style="opacity: {loading ? 0.5 : 1}">
  {#each current as video (video.url)}
    <div style="display:flex">
      <div class="video" on:click={() => window.open(video.url)}>
        <img src={video.image} alt={video.name} width="150px" height="208px" />
        {video.name}
      </div>
      <div class="playlist">
        {#if video.playlist}
          <Playlist bind:name={video.name} bind:playlist={video.playlist} />
        {/if}
      </div>
    </div>
  {/each}
</div>
<div class="loading" hidden={!loading}>
  <div class="sk-wave sk-center">
    <div class="sk-wave-rect" />
    <div class="sk-wave-rect" />
    <div class="sk-wave-rect" />
    <div class="sk-wave-rect" />
    <div class="sk-wave-rect" />
  </div>
</div>

<style>
  :root {
    --sk-color: #1a73e8;
    --header: 80px;
  }

  header {
    padding: 10px 20px;
  }

  .navbar {
    user-select: none;
    height: var(--header);
    justify-content: space-between;
    letter-spacing: 0.3px;
    border-bottom: 5px solid #f2f2f2;
  }

  .navbar-brand {
    font-size: 24px;
    padding-left: 30px;
  }

  .input-group {
    width: 33%;
    max-width: 360px;
  }

  .input-group > :not(:last-child) {
    margin-right: -1px;
    border-top-right-radius: 0;
    border-bottom-right-radius: 0;
  }

  .input-group > :not(:first-child) {
    border-top-left-radius: 0;
    border-bottom-left-radius: 0;
    z-index: 2;
  }

  svg {
    vertical-align: -0.125em;
  }

  .content,
  .loading {
    position: fixed;
    top: var(--header);
    width: 100%;
    height: calc(100% - var(--header));
  }

  .content {
    overflow: auto;
  }

  .loading {
    z-index: 2;
    display: flex;
  }

  .video {
    display: grid;
    margin: 10px;
    text-align: center;
    width: 150px;
    cursor: pointer;
  }

  .playlist {
    height: 230px;
    width: calc(100% - 170px);
    overflow: auto;
    align-self: center;
  }

  @media (max-width: 767px) {
    :root {
      --header: 160px;
    }

    .navbar {
      border-color: transparent;
    }

    .navbar-brand {
      padding-left: 0;
    }

    .input-group {
      width: 66%;
    }
  }

  :global(body) {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
      "Helvetica Neue", Arial, "Noto Sans", "Microsoft YaHei New",
      "Microsoft Yahei", 微软雅黑, 宋体, SimSun, STXihei, 华文细黑, sans-serif;
  }
</style>
