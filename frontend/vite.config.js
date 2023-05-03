import vue from '@vitejs/plugin-vue';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [vue()],
	root: 'src/',
	base: './',
	resolve: {
		dedupe: ['vue'],
	},
	build: {
		outDir: '../dist',
	},
	server: {
		host: '127.0.0.1',
		port: 9272,
	},
});
