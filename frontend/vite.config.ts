import { defineConfig } from 'vite';
import { resolve } from 'path';

export default defineConfig({
  build: {
    outDir: '../internal/handlers/assets',
    emptyOutDir: false, // Don't delete images directory
    sourcemap: true,
    minify: 'terser',
    rollupOptions: {
      input: {
        home: resolve(__dirname, 'src/home.ts'),
        settings: resolve(__dirname, 'src/settings.ts'),
        // CSS entry point
        styles: resolve(__dirname, '../internal/handlers/assets/css/input.css'),
      },
      output: {
        entryFileNames: 'js/[name].js',
        chunkFileNames: 'js/[name].js',
        assetFileNames: (assetInfo) => {
          // Place CSS in css directory
          if (assetInfo.name?.endsWith('.css')) {
            return 'css/[name][extname]';
          }
          return '[name][extname]';
        },
      },
    },
  },
});
