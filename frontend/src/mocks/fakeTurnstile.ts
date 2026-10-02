import { act, waitFor } from '@testing-library/react'
import { expect } from 'vitest'

type TurnstileParams = {
  sitekey: string
  callback: (token: string) => void
  'expired-callback': () => void
}

type WindowTurnstile = NonNullable<Window['turnstile']>

// jsdom cannot run Cloudflare's widget, so tests stand in for it at the
// window.turnstile boundary and hand tokens to the component the way the
// real widget does.
export class FakeTurnstile {
  params: TurnstileParams | null = null
  resets = 0

  install() {
    const fake = {
      render: (_container: unknown, params: TurnstileParams) => {
        this.params = params
        return 'fake-widget'
      },
      reset: () => {
        this.resets += 1
      },
      remove: () => {},
    }
    window.turnstile = fake as unknown as WindowTurnstile
  }

  async waitForWidget() {
    await waitFor(() => expect(this.params).not.toBeNull())
  }

  async solve(token: string) {
    await this.waitForWidget()
    act(() => this.params?.callback(token))
  }

  async expire() {
    await this.waitForWidget()
    act(() => this.params?.['expired-callback']())
  }
}
