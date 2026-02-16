import { mount } from 'svelte'
import './app.css'
import AppFinal from './AppFinal.svelte'

const app = mount(AppFinal, {
  target: document.getElementById('app'),
})

export default app
