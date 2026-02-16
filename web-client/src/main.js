import { mount } from 'svelte'
import './app.css'
import AppYazi from './AppYazi.svelte'

const app = mount(AppYazi, {
  target: document.getElementById('app'),
})

export default app
