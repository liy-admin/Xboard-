import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import fs from 'node:fs';
import path from 'node:path';

const languageWorkerNames = [
  'css.worker.js',
  'html.worker.js',
  'json.worker.js',
  'ts.worker.js',
];

const editorWorkerName = 'editor.worker.js';
const monacoWorkerNames = [...languageWorkerNames, editorWorkerName];

function monacoWorkerAssets() {
  const sourceDir = path.resolve(__dirname, '../vendor/monaco-workers');
  const outputDir = path.resolve(__dirname, '../../public/assets/admin/assets/workers');

  function createEditorWorkerSource() {
    const jsonWorkerSource = fs.readFileSync(path.join(sourceDir, 'json.worker.js'), 'utf8');
    const bootstrapIndex = jsonWorkerSource.lastIndexOf('    self.onmessage = () => {');

    if (bootstrapIndex < 0) {
      throw new Error('Cannot find Monaco worker bootstrap in json.worker.js');
    }

    // Reuse the original bundled core worker, but skip the JSON language bootstrap.
    return `${jsonWorkerSource.slice(0, bootstrapIndex)}    self.onmessage = () => {
        ns(() => undefined);
    };
})();
`;
  }

  function writeWorkerAsset(filename, destination) {
    if (filename === editorWorkerName) {
      fs.writeFileSync(destination, createEditorWorkerSource());
      return;
    }

    fs.copyFileSync(path.join(sourceDir, filename), destination);
  }

  return {
    name: 'xboard-monaco-worker-assets',
    configureServer(server) {
      server.middlewares.use('/assets/workers', (req, res, next) => {
        const filename = path.basename(req.url.split('?')[0]);
        if (!monacoWorkerNames.includes(filename)) {
          next();
          return;
        }

        const workerPath = path.join(sourceDir, filename);
        if (filename !== editorWorkerName && !fs.existsSync(workerPath)) {
          next();
          return;
        }

        res.setHeader('Content-Type', 'text/javascript; charset=utf-8');
        if (filename === editorWorkerName) {
          res.end(createEditorWorkerSource());
          return;
        }

        fs.createReadStream(workerPath).pipe(res);
      });
    },
    closeBundle() {
      fs.mkdirSync(outputDir, { recursive: true });
      for (const filename of monacoWorkerNames) {
        writeWorkerAsset(filename, path.join(outputDir, filename));
      }
    },
  };
}

function manifestRelocate() {
  const outDir = path.resolve(__dirname, '../../public/assets/admin');
  const viteDir = path.join(outDir, '.vite');

  return {
    name: 'xboard-manifest-relocate',
    closeBundle() {
      const generated = path.join(viteDir, 'manifest.json');
      if (!fs.existsSync(generated)) return;

      // admin.blade.php reads assets/admin/manifest.json directly (pre-Vite-5 layout).
      fs.copyFileSync(generated, path.join(outDir, 'manifest.json'));
      fs.rmSync(viteDir, { recursive: true, force: true });
    },
  };
}

function devSettings() {
  return {
    name: 'xboard-dev-settings',
    configureServer(server) {
      server.middlewares.use((req, res, next) => {
        const url = req.url.split('?')[0];
        if (url !== '/settings.js' && url !== '/settings.local.js') {
          next();
          return;
        }

        let cfg = {};
        try {
          const raw = fs.readFileSync(path.resolve(__dirname, 'config.json'), 'utf8');
          cfg = JSON.parse(raw).setting || {};
        } catch {
          // no config.json — serve empty
        }

        const baseUrl = cfg.appDomain || '';
        const securePath = (cfg.appAdminPath || '').replace(/^\/+|\/+$/g, '');

        res.setHeader('Content-Type', 'text/javascript; charset=utf-8');
        res.end(
          `window.settings = window.settings || {};\n` +
          (baseUrl ? `window.settings.base_url = ${JSON.stringify(baseUrl)};\n` : '') +
          (securePath ? `window.settings.secure_path = ${JSON.stringify(securePath)};\n` : '')
        );
      });
    },
  };
}

export default defineConfig({
  plugins: [react(), monacoWorkerAssets(), manifestRelocate(), devSettings()],
  base: './',
  server: {
    // dev 下页面在 5173，接口走相对路径，由这里转给本地 Go 面板（它不处理跨域请求）
    proxy: {
      // 只转接口前缀；不能写 '/api'，源码目录 api/（api/config.js 等）在 dev 下的地址也以它开头
      '/api/v1': { target: process.env.XBOARD_DEV_API || 'http://127.0.0.1:7001', changeOrigin: true },
      '/api/v2': { target: process.env.XBOARD_DEV_API || 'http://127.0.0.1:7001', changeOrigin: true },
      '/ws': { target: (process.env.XBOARD_DEV_API || 'http://127.0.0.1:7001').replace(/^http/, 'ws'), ws: true },
    },
  },
  esbuild: {
    loader: 'jsx',
    include: /source\/.*\.jsx?$/,
    exclude: [],
  },
  optimizeDeps: {
    esbuildOptions: {
      loader: {
        '.js': 'jsx',
      },
    },
  },
  build: {
    outDir: '../../public/assets/admin',
    emptyOutDir: true,
    manifest: true,
    rollupOptions: {
      output: {
        assetFileNames: 'assets/[name]-[hash][extname]',
        chunkFileNames: 'assets/[name]-[hash].js',
        entryFileNames: 'assets/[name]-[hash].js',
      }
    }
  }
});
