import { defineConfig } from '@playwright/test'
export default defineConfig({
 testDir:'./e2e',workers:1,timeout:45000,
 use:{baseURL:'http://127.0.0.1:18892',launchOptions:{executablePath:process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE},screenshot:'only-on-failure'},
 webServer:{command:'npm run preview -- --host 127.0.0.1 --port 18892',url:'http://127.0.0.1:18892',reuseExistingServer:false}
})
