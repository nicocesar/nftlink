import type { PlaywrightTestConfig } from '@playwright/test';

const config: PlaywrightTestConfig = {
	webServer: {
		command: 'npm run build && npm run preview',
		port: 3000
	},
	timeout: 2000,
	fullyParallel: true
	
};

export default config;
