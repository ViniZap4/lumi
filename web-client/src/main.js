import { mount } from 'svelte'
import './app.css'
import AppSimple2 from './AppSimple2.svelte'

const app = mount(AppSimple2, {
  target: document.getElementById('app'),
})

export default app
