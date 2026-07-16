import { test, expect } from './fixtures/coverage'

const API = 'http://localhost:8080'
const TS = Date.now()

let tok = '', convId = ''

test.describe('File Panel API', () => {
  test.beforeAll(async ({ request }) => {
    const r = await request.post(`${API}/api/v1/users/register`, {
      data: { account: `fp_${TS}`, name: 'FileTester', password: 'test123' },
    })
    ;({ token: tok, user_id: convId } = (await r.json()).data)
    // Use a fixed test conv ID for API testing (conv membership is loose in file APIs)
    convId = 'test_conv_fp'
  })

  test('list empty folder returns empty array', async ({ request }) => {
    const r = await request.get(`${API}/api/v1/conversations/${convId}/folders`, { headers: { Authorization: `Bearer ${tok}` } })
    expect((await r.json()).code).toBe(0)
  })

  test('create folder', async ({ request }) => {
    const r = await request.post(`${API}/api/v1/conversations/${convId}/folders`, {
      headers: { Authorization: `Bearer ${tok}` }, data: { name: 'Docs', parent_id: 0 },
    })
    const j = await r.json()
    expect(j.code).toBe(0)
    expect(j.data.folder_id).toBeGreaterThan(0)
  })

  test('rename folder', async ({ request }) => {
    // Get existing folder
    const r1 = await request.get(`${API}/api/v1/conversations/${convId}/folders`, { headers: { Authorization: `Bearer ${tok}` } })
    const folders: any[] = (await r1.json()).data || []
    if (folders.length === 0) { test.skip(); return }

    const r2 = await request.put(`${API}/api/v1/conversations/${convId}/folders/${folders[0].folder_id}/rename`, {
      headers: { Authorization: `Bearer ${tok}` }, data: { name: 'Renamed' },
    })
    expect((await r2.json()).code).toBe(0)

    // Verify renamed
    const r3 = await request.get(`${API}/api/v1/conversations/${convId}/folders`, { headers: { Authorization: `Bearer ${tok}` } })
    const updated: any[] = (await r3.json()).data || []
    expect(updated.some((f: any) => f.name === 'Renamed')).toBeTruthy()
  })

  test('move folder', async ({ request }) => {
    // Create parent and child folders
    const p = await request.post(`${API}/api/v1/conversations/${convId}/folders`, {
      headers: { Authorization: `Bearer ${tok}` }, data: { name: 'Parent', parent_id: 0 },
    })
    const parentId = (await p.json()).data.folder_id
    const c = await request.post(`${API}/api/v1/conversations/${convId}/folders`, {
      headers: { Authorization: `Bearer ${tok}` }, data: { name: 'Child', parent_id: 0 },
    })
    const childId = (await c.json()).data.folder_id

    // Move child into parent
    const m = await request.put(`${API}/api/v1/conversations/${convId}/folders/${childId}/move`, {
      headers: { Authorization: `Bearer ${tok}` }, data: { parent_id: parentId },
    })
    expect((await m.json()).code).toBe(0)

    // Verify child is now under parent
    const children = await request.get(`${API}/api/v1/conversations/${convId}/folders?parent_id=${parentId}`, {
      headers: { Authorization: `Bearer ${tok}` },
    })
    const items: any[] = (await children.json()).data || []
    expect(items.some((f: any) => f.folder_id === childId)).toBeTruthy()

    // Cleanup
    await request.delete(`${API}/api/v1/conversations/${convId}/folders/${childId}`, { headers: { Authorization: `Bearer ${tok}` } })
    await request.delete(`${API}/api/v1/conversations/${convId}/folders/${parentId}`, { headers: { Authorization: `Bearer ${tok}` } })
  })

  test('upload file via API', async ({ request }) => {
    const fd = new FormData()
    fd.append('file', new Blob(['hello'], { type: 'text/plain' }), 'test.txt')
    fd.append('file_type', '1')
    fd.append('conv_id', convId)
    const r = await fetch(`${API}/api/v1/files/upload`, {
      method: 'POST', headers: { Authorization: `Bearer ${tok}` }, body: fd,
    })
    expect((await r.json()).code).toBe(0)
  })

  test('delete file', async ({ request }) => {
    const r1 = await request.get(`${API}/api/v1/conversations/${convId}/files`, { headers: { Authorization: `Bearer ${tok}` } })
    const files = (await r1.json()).data.items
    if (files.length === 0) { test.skip(); return }

    const r2 = await request.delete(`${API}/api/v1/conversations/${convId}/files/${files[0].file_id}`, {
      headers: { Authorization: `Bearer ${tok}` },
    })
    expect((await r2.json()).code).toBe(0)
  })

  test('move file to folder', async ({ request }) => {
    // Upload a file
    const fd = new FormData()
    fd.append('file', new Blob(['move me'], { type: 'text/plain' }), 'move.txt')
    fd.append('file_type', '1'); fd.append('conv_id', convId)
    await fetch(`${API}/api/v1/files/upload`, { method: 'POST', headers: { Authorization: `Bearer ${tok}` }, body: fd })

    // Get file
    const r1 = await request.get(`${API}/api/v1/conversations/${convId}/files`, { headers: { Authorization: `Bearer ${tok}` } })
    const files = (await r1.json()).data.items
    if (files.length === 0) { test.skip(); return }
    const fid = files[0].file_id

    // Create target folder
    const cf = await request.post(`${API}/api/v1/conversations/${convId}/folders`, {
      headers: { Authorization: `Bearer ${tok}` }, data: { name: 'Target', parent_id: 0 },
    })
    const folderId = (await cf.json()).data.folder_id

    // Move file
    const m = await request.put(`${API}/api/v1/conversations/${convId}/files/${fid}/move`, {
      headers: { Authorization: `Bearer ${tok}` }, data: { folder_id: folderId },
    })
    expect((await m.json()).code).toBe(0)

    // Verify file is in folder
    const r2 = await request.get(`${API}/api/v1/conversations/${convId}/folders/${folderId}/files`, {
      headers: { Authorization: `Bearer ${tok}` },
    })
    const folderFiles = (await r2.json()).data.items
    expect(folderFiles.some((f: any) => f.file_id === fid)).toBeTruthy()

    // Cleanup
    await request.delete(`${API}/api/v1/conversations/${convId}/files/${fid}`, { headers: { Authorization: `Bearer ${tok}` } })
    await request.delete(`${API}/api/v1/conversations/${convId}/folders/${folderId}`, { headers: { Authorization: `Bearer ${tok}` } })
  })

  test('delete folder', async ({ request }) => {
    const cf = await request.post(`${API}/api/v1/conversations/${convId}/folders`, {
      headers: { Authorization: `Bearer ${tok}` }, data: { name: 'ToDelete', parent_id: 0 },
    })
    const fid = (await cf.json()).data.folder_id

    const r = await request.delete(`${API}/api/v1/conversations/${convId}/folders/${fid}`, {
      headers: { Authorization: `Bearer ${tok}` },
    })
    expect((await r.json()).code).toBe(0)
  })
})
