import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const USE_API_FETCH = resolve(__dirname, '../../composables/useApiFetch.ts')
const source = () => readFileSync(USE_API_FETCH, 'utf8')

describe('useApiFetch error normalization', () => {
  it('reads HTTP failures from the ofetch response context', () => {
    expect(source()).toMatch(/onResponseError\(\{ response \}/)
    expect(source()).not.toMatch(/onResponseError\(\{ error \}/)
  })

  it('never throws an undefined response error', () => {
    const contents = source()
    expect(contents).toMatch(/new Error\(message\)/)
    expect(contents).toMatch(/onRequestError\(\{ error \}/)
  })
})
