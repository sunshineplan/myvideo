import sveltePreprocess from 'svelte-preprocess'

export default {
  // Consult https://github.com/sveltejs/svelte-preprocess
  // for more information about preprocessors
  preprocess: sveltePreprocess(),
  compilerOptions: { cssHash: ({ hash, css }) => { return `s${hash(css)}` } }
}
