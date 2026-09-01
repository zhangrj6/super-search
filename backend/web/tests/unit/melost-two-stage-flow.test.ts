import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const read = (path: string) => readFileSync(resolve(__dirname, path), 'utf-8')

describe('melost two-stage resource flow', () => {
  it('search results stage a resource and do not expose a link action', () => {
    const component = read('../../components/MelostSearchResults.vue')
    const composable = read('../../composables/useMelostApi.ts')

    expect(component).not.toContain('获取链接')
    expect(component).toContain('stageAndOpen(item)')
    expect(component).toContain('router.push(`/r/${result.resource_key}`)')
    expect(composable).toContain("useApiFetch('/resources/stage'")
    expect(composable).not.toContain('/melost')
    expect(composable).not.toContain('getImportStatus')
  })

  it('detail failures clear both link fields', () => {
    const detailPage = read('../../pages/r/[key].vue')
    const catchStart = detailPage.indexOf("console.error('获取资源链接失败:'")
    const catchEnd = detailPage.indexOf('} finally {', catchStart)
    const catchBody = detailPage.slice(catchStart, catchEnd)

    expect(catchBody).toContain("url: ''")
    expect(catchBody).toContain("save_url: ''")
  })

  it('error modal has no link, open, or copy controls', () => {
    const modal = read('../../components/QrCodeModal.vue')
    const errorStart = modal.indexOf('v-else-if="error"')
    const errorEnd = modal.indexOf('v-else-if="deliveryUrl"', errorStart)
    const errorBlock = modal.slice(errorStart, errorEnd)

    expect(errorBlock).not.toContain('{{ deliveryUrl }}')
    expect(errorBlock).not.toContain('@click="openLink"')
    expect(errorBlock).not.toContain('@click="copyUrl"')
  })
})
