import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				// Mode runes (Svelte 5) partout sauf node_modules.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},

			// SPA statique : servie par nginx (prod) ou vite (dev),
			// l'API Go reste la seule source de vérité.
			adapter: adapter({ fallback: 'index.html' })
		})
	],

	server: {
		// Dev : l'API locale tourne sur :8080 (pas de CORS nécessaire).
		proxy: {
			'/v1': 'http://localhost:8080',
			'/healthz': 'http://localhost:8080'
		}
	}
});
