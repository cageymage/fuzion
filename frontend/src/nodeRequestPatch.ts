// Data-router navigation builds a Request with jsdom's AbortSignal, which Node's
// fetch Request rejects. Tests never abort a Request, so drop the signal. Imported
// before MSW so its interceptors wrap the patched Request.
const NodeRequest = globalThis.Request

globalThis.Request = class extends NodeRequest {
  constructor(input: RequestInfo | URL, init?: RequestInit) {
    super(input, init?.signal ? { ...init, signal: undefined } : init)
  }
}
