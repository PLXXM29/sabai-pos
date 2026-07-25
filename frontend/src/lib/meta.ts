// Which deployment am I? The bundle is built once and can be served by a shop's
// own install or by the public demo, so the difference has to be asked for at
// runtime. Fetched once per page load and shared, since the answer only changes
// when the server is redeployed.
import { useEffect, useState } from 'react'
import { api, type DeploymentMeta } from './api'

// Assume "a real shop" when the server cannot be reached: the offline-first
// cashier still has to work on a dead network, and the safe default is to show
// no demo affordances rather than to invent them.
const OFFLINE_FALLBACK: DeploymentMeta = { version: 'unknown', demo: false }

let inflight: Promise<DeploymentMeta> | null = null

export function loadMeta(): Promise<DeploymentMeta> {
  inflight ??= api.meta().catch(() => OFFLINE_FALLBACK)
  return inflight
}

/** Null until the answer arrives — render neutrally rather than guessing. */
export function useMeta(): DeploymentMeta | null {
  const [meta, setMeta] = useState<DeploymentMeta | null>(null)
  useEffect(() => {
    let alive = true
    loadMeta().then((m) => {
      if (alive) setMeta(m)
    })
    return () => {
      alive = false
    }
  }, [])
  return meta
}
