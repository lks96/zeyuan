import { createApp } from 'vue'
import { createVuestic } from 'vuestic-ui'
import 'vuestic-ui/css'

import App from './App.vue'
import router from './router'
import './styles/main.css'

createApp(App)
  .use(router)
  .use(createVuestic({
    config: {
      colors: {
        variables: {
          primary: '#2563eb',
          secondary: '#0f766e',
          success: '#16a34a',
          warning: '#d97706',
          danger: '#dc2626',
          backgroundPrimary: '#f6f7fb',
          backgroundSecondary: '#ffffff',
          textPrimary: '#1f2937',
        },
      },
    },
  }))
  .mount('#app')
