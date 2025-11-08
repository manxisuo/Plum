import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
	plugins: [vue()],
	server: {
		host: '0.0.0.0',  // 监听所有网络接口，允许外部访问
		port: 5173,
		proxy: {
			'/v1': {
				target: 'http://plum-controller:8080',
				changeOrigin: true
			}
		}
	}
})


