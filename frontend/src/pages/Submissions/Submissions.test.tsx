import { fireEvent, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it, vi } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import type { AuthUser } from '../../types/auth'
import { Submissions } from './Submissions'

const member: AuthUser = {
  id: 'user-1',
  username: 'thundermane',
  avatarUrl: null,
  isOfficer: false,
  isGuildMember: true,
  battlenetLinked: false,
  battletag: null,
}

function png(name: string): File {
  return new File(['png-bytes'], name, { type: 'image/png' })
}

function asMember(user: AuthUser = member) {
  server.use(http.get('/api/auth/me', () => HttpResponse.json(user)))
}

async function chooseFiles(...files: File[]) {
  await userEvent.upload(await screen.findByLabelText('Images to submit'), files)
}

// jsdom's Blob has no text(), but it does implement FileReader.
function readText(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result as string)
    reader.onerror = () => reject(reader.error)
    reader.readAsText(blob)
  })
}

function selectedNames() {
  const list = screen.queryByRole('list', { name: 'Selected images' })
  if (!list) return []
  return within(list)
    .getAllByRole('listitem')
    .map((item) => within(item).getByRole('img').getAttribute('alt'))
}

describe('Submissions', () => {
  it('should show a closed message when submissions are temporarily closed', async () => {
    asMember()
    server.use(http.get('/api/submissions/status', () => HttpResponse.json({ open: false, attestationVersion: '2026-10-26' })))

    renderWithProviders(<Submissions />, '/submissions')

    expect(await screen.findByText('Submissions are temporarily closed. Please try again later.')).toBeInTheDocument()
    expect(screen.queryByRole('form', { name: 'Submit art or screenshots' })).not.toBeInTheDocument()
  })

  it('should prompt to log in when the visitor is logged out', async () => {
    renderWithProviders(<Submissions />, '/submissions')

    expect(await screen.findByRole('link', { name: 'Log in with Discord' })).toHaveAttribute(
      'href',
      'http://localhost:3000/api/auth/login?returnTo=%2Fsubmissions',
    )
  })

  it("should say only guild members can submit when the user is not in the guild's Discord server", async () => {
    asMember({ ...member, isGuildMember: false })

    renderWithProviders(<Submissions />, '/submissions')

    expect(
      await screen.findByText("Only members of the guild's Discord server can submit images."),
    ).toBeInTheDocument()
  })

  it('should list each chosen image with a preview when files are picked with the browse button', async () => {
    asMember()
    renderWithProviders(<Submissions />, '/submissions')

    await chooseFiles(png('onyxia.png'), png('raid.png'))

    expect(selectedNames()).toEqual(['Preview of onyxia.png', 'Preview of raid.png'])
    expect(screen.getByRole('img', { name: 'Preview of onyxia.png' })).toHaveAttribute('src', 'blob:preview/onyxia.png')
  })

  it('should add images dropped on the drop zone', async () => {
    asMember()
    renderWithProviders(<Submissions />, '/submissions')

    fireEvent.drop(await screen.findByTestId('dropzone'), { dataTransfer: { files: [png('dropped.png')] } })

    expect(selectedNames()).toEqual(['Preview of dropped.png'])
  })

  it('should take an image off the list when its Remove button is clicked', async () => {
    asMember()
    renderWithProviders(<Submissions />, '/submissions')
    await chooseFiles(png('keep.png'), png('drop.png'))

    await userEvent.click(screen.getByRole('button', { name: 'Remove drop.png' }))

    expect(selectedNames()).toEqual(['Preview of keep.png'])
  })

  it('should keep Submit disabled until the attestation is checked', async () => {
    asMember()
    renderWithProviders(<Submissions />, '/submissions')
    await chooseFiles(png('shot.png'))
    const submit = screen.getByRole('button', { name: 'Submit' })
    expect(submit).toBeDisabled()

    await userEvent.click(screen.getByRole('checkbox', { name: /I created these images or bought the rights to them/ }))

    expect(submit).toBeEnabled()
  })

  it('should refuse a file that is not a supported image before uploading', async () => {
    asMember()
    renderWithProviders(<Submissions />, '/submissions')

    // applyAccept off: a browser's picker can still be switched to "All files".
    await userEvent.upload(await screen.findByLabelText('Images to submit'), new File(['x'], 'notes.txt', { type: 'text/plain' }), {
      applyAccept: false,
    })

    expect(screen.getByRole('alert')).toHaveTextContent('notes.txt: must be a PNG, JPEG, WebP or GIF image')
    expect(selectedNames()).toEqual([])
  })

  it('should refuse a file over 20 MiB before uploading', async () => {
    asMember()
    renderWithProviders(<Submissions />, '/submissions')
    const huge = png('huge.png')
    Object.defineProperty(huge, 'size', { value: 20 * 1024 * 1024 + 1 })

    await chooseFiles(huge)

    expect(screen.getByRole('alert')).toHaveTextContent('huge.png: must be at most 20 MiB')
    expect(selectedNames()).toEqual([])
  })

  it('should refuse an eleventh image in one submission', async () => {
    asMember()
    renderWithProviders(<Submissions />, '/submissions')

    await chooseFiles(...Array.from({ length: 11 }, (_, i) => png(`shot-${i}.png`)))

    expect(screen.getByRole('alert')).toHaveTextContent('You can add at most 10 images per submission.')
    expect(selectedNames()).toHaveLength(10)
  })

  it('should send the images with their descriptions and show a result for each when Submit is clicked', async () => {
    asMember()
    // jsdom's FormData is not one Node's fetch can serialize, so the mocked request has no
    // readable body; this records what the page put in the form instead.
    const appended = vi.spyOn(FormData.prototype, 'append')
    server.use(
      http.post('/api/submissions', () => {
        return HttpResponse.json({
          results: [
            { filename: 'onyxia.png', submission: { id: 'sub-1' } },
            { filename: 'again.png', error: 'you have already submitted this image' },
          ],
        }, { status: 201 })
      }),
    )
    renderWithProviders(<Submissions />, '/submissions')
    await chooseFiles(png('onyxia.png'), png('again.png'))
    await userEvent.type(screen.getByRole('textbox', { name: 'Description for onyxia.png' }), 'Onyxia down!')
    await userEvent.click(screen.getByRole('checkbox', { name: /I created these images/ }))

    await userEvent.click(screen.getByRole('button', { name: 'Submit' }))

    const results = await screen.findByRole('list', { name: 'Submission results' })
    expect(within(results).getAllByRole('listitem').map((item) => item.textContent)).toEqual([
      'onyxia.png: submitted',
      'again.png: you have already submitted this image',
    ])
    const [metadataCall, ...fileCalls] = appended.mock.calls
    appended.mockRestore()
    expect(metadataCall[0]).toBe('metadata')
    expect(JSON.parse(await readText(metadataCall[1] as Blob))).toEqual({
      attestationAccepted: true,
      attestationVersion: '2026-10-26',
      descriptions: ['Onyxia down!', ''],
    })
    expect(fileCalls.map(([field, , filename]) => [field, filename])).toEqual([
      ['files', 'onyxia.png'],
      ['files', 'again.png'],
    ])
    expect(selectedNames()).toEqual(['Preview of again.png'])
  })

  it("should show the server's message when the daily quota is used up", async () => {
    asMember()
    server.use(
      http.post('/api/submissions', () =>
        HttpResponse.json({ error: 'you can submit at most 20 images per 24 hours; 0 left' }, { status: 429 }),
      ),
    )
    renderWithProviders(<Submissions />, '/submissions')
    await chooseFiles(png('shot.png'))
    await userEvent.click(screen.getByRole('checkbox', { name: /I created these images/ }))

    await userEvent.click(screen.getByRole('button', { name: 'Submit' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('you can submit at most 20 images per 24 hours; 0 left')
    expect(selectedNames()).toEqual(['Preview of shot.png'])
  })

  it('should show the license grant on the form', async () => {
    asMember()

    renderWithProviders(<Submissions />, '/submissions')

    const license = await screen.findByRole('region', { name: 'License' })
    expect(license).toHaveTextContent(
      'you grant the Fuzion guild an unrestricted license to use these images on the guild site and in any related social media posts',
    )
  })

  it('should link officers to the review page', async () => {
    asMember({ ...member, isOfficer: true })

    renderWithProviders(<Submissions />, '/submissions')

    expect(await screen.findByRole('link', { name: 'Review submissions' })).toHaveAttribute('href', '/officer/submissions')
  })
})
