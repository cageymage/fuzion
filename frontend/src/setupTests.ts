import './nodeRequestPatch'
import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterAll, afterEach, beforeAll } from 'vitest'
import { server } from './mocks/server'

// jsdom does not implement showModal() or close() on <dialog>.
HTMLDialogElement.prototype.showModal = function showModal() {
  this.setAttribute('open', '')
}
HTMLDialogElement.prototype.close = function close() {
  this.removeAttribute('open')
  this.dispatchEvent(new Event('close'))
}

// jsdom has no object URLs; previews only need a string to put in src.
URL.createObjectURL = (blob: Blob) => `blob:preview/${blob instanceof File ? blob.name : 'blob'}`
URL.revokeObjectURL = () => {}

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))

afterEach(() => {
  cleanup()
  server.resetHandlers()
})

afterAll(() => server.close())
