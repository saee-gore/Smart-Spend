#!/usr/bin/env node
'use strict';

const http = require('http');
const fs   = require('fs');
const path = require('path');
const { exec } = require('child_process');

const PORT   = process.env.PORT || 3000;
const PUBLIC = path.join(__dirname, 'public');

const MIME = {
  '.html': 'text/html',
  '.css':  'text/css',
  '.js':   'application/javascript',
  '.json': 'application/json',
  '.png':  'image/png',
  '.jpg':  'image/jpeg',
  '.svg':  'image/svg+xml',
  '.ico':  'image/x-icon',
};

const server = http.createServer((req, res) => {
  // Normalise URL → file path
  let urlPath = req.url.split('?')[0];
  if (urlPath === '/') urlPath = '/index.html';

  const filePath = path.join(PUBLIC, urlPath);

  // Security: prevent directory traversal
  if (!filePath.startsWith(PUBLIC)) {
    res.writeHead(403);
    return res.end('Forbidden');
  }

  fs.readFile(filePath, (err, data) => {
    if (err) {
      // Fall back to index.html for any missing route (SPA behaviour)
      fs.readFile(path.join(PUBLIC, 'index.html'), (err2, html) => {
        if (err2) { res.writeHead(404); return res.end('Not found'); }
        res.writeHead(200, { 'Content-Type': 'text/html' });
        res.end(html);
      });
      return;
    }
    const ext  = path.extname(filePath).toLowerCase();
    const mime = MIME[ext] || 'application/octet-stream';
    res.writeHead(200, { 'Content-Type': mime });
    res.end(data);
  });
});

server.listen(PORT, () => {
  const url = `http://localhost:${PORT}`;

  console.log('');
  console.log('\x1b[38;5;135m\x1b[1m  ✦ Smart Spend Dashboard\x1b[0m');
  console.log('\x1b[2m  Privacy-first · all processing happens in your browser\x1b[0m');
  console.log('');
  console.log(`\x1b[32m  ✓  Running at \x1b[36m${url}\x1b[0m`);
  console.log('\x1b[2m  Press Ctrl+C to stop\x1b[0m');
  console.log('');

  // Auto-open browser
  const opener =
    process.platform === 'darwin' ? `open "${url}"` :
    process.platform === 'win32'  ? `start "" "${url}"` :
                                    `xdg-open "${url}"`;
  exec(opener, err => {
    if (err) console.log(`\x1b[33m  → Open your browser and navigate to ${url}\x1b[0m\n`);
  });
});

server.on('error', err => {
  if (err.code === 'EADDRINUSE') {
    console.error(`\n  \x1b[31m✗  Port ${PORT} is already in use.\x1b[0m`);
    console.error(`  Try:  PORT=3001 npm start\n`);
  } else {
    console.error(err);
  }
  process.exit(1);
});
