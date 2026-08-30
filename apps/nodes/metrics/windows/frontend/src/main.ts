import './style.css'
import { wailsMetricsAPI } from './api'
import { createMetricsApp } from './app'

const root = document.querySelector<HTMLElement>('#app')
if (!root) throw new Error('Metrics UI root #app 不存在')

const app = createMetricsApp(root, wailsMetricsAPI)
window.addEventListener('beforeunload', app.dispose, {once: true})
