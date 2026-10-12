package web

import _ "embed"

const manifest = `{"name":"My Gym","short_name":"My Gym","start_url":"/workouts","scope":"/","display":"standalone","background_color":"#f3f3f3","theme_color":"#222222","icons":[{"src":"/icon.svg","sizes":"any","type":"image/svg+xml","purpose":"any maskable"}]}`
const serviceWorker = `const CACHE='gym-shell-v1';const ASSETS=['/icon.svg','/manifest.webmanifest'];self.addEventListener('install',e=>e.waitUntil(caches.open(CACHE).then(c=>c.addAll(ASSETS))));self.addEventListener('activate',e=>e.waitUntil(caches.keys().then(keys=>Promise.all(keys.filter(k=>k!==CACHE).map(k=>caches.delete(k))))));self.addEventListener('fetch',e=>{const u=new URL(e.request.url);if(e.request.method!=='GET')return;if(u.origin!==self.location.origin)return;if(ASSETS.includes(u.pathname))e.respondWith(caches.match(e.request).then(r=>r||fetch(e.request)));});`

//go:embed templates/icon.svg
var icon []byte
