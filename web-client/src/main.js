import { mount } from 'svelte'
import './app.css'
import AppClean from './AppClean.svelte'

const app = mount(AppClean, {
  target: document.getElementById('app'),
})

export default app
