import assert from 'node:assert/strict'
import test from 'node:test'
import {
  UPLOAD_TIMEOUT_FLOOR_MS,
  isTimeoutError,
  uploadPayloadBytes,
  uploadTimeoutMs,
} from './requestTimeouts'

const formDataWith = (...sizes: number[]) => {
  const form = new FormData()
  form.append('name', 'not-a-blob')
  sizes.forEach((size, i) => form.append(`file${i}`, new Blob([new Uint8Array(size)])))
  return form
}

test('payload size counts blobs and ignores plain fields', () => {
  assert.equal(uploadPayloadBytes(formDataWith()), 0)
  assert.equal(uploadPayloadBytes(formDataWith(1024)), 1024)
  assert.equal(uploadPayloadBytes(formDataWith(1024, 2048)), 3072)
})

test('non-FormData payloads contribute no bytes', () => {
  assert.equal(uploadPayloadBytes({ file: 'x' }), 0)
  assert.equal(uploadPayloadBytes(undefined), 0)
})

test('small uploads still get the floor', () => {
  assert.equal(uploadTimeoutMs(formDataWith(1024)), UPLOAD_TIMEOUT_FLOOR_MS)
})

test('large uploads scale past the floor', () => {
  assert.equal(uploadTimeoutMs(formDataWith(100 * 1024 * 1024)), UPLOAD_TIMEOUT_FLOOR_MS)
  assert.equal(uploadTimeoutMs(formDataWith(700 * 1024 * 1024)), 70 * 60 * 1000)
})

test('timeouts are distinguished from unreachable-server failures', () => {
  assert.equal(isTimeoutError({ code: 'ECONNABORTED', message: 'timeout of 30000ms exceeded' }), true)
  assert.equal(isTimeoutError({ code: 'ETIMEDOUT' }), true)
  assert.equal(isTimeoutError({ message: 'timeout exceeded' }), true)
  assert.equal(isTimeoutError({ code: 'ERR_NETWORK', message: 'Network Error' }), false)
  assert.equal(isTimeoutError(null), false)
})
