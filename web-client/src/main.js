import { mount } from 'svelte'
import './app.css'
import AppSimple from './AppSimple.svelte'

const app = mount(AppSimple, {
  target: document.getElementById('app'),
})

export default app
