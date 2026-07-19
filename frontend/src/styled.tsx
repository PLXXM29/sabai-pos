import {
  useMemo,
  useState,
  createElement,
  type CSSProperties,
  type ReactNode,
} from 'react'

/**
 * Convert a raw CSS declaration string ("display:flex;gap:8px") into a React
 * style object. This lets us transcribe the original design's inline styles
 * verbatim instead of hand-camelCasing every rule.
 */
const cache = new Map<string, CSSProperties>()

export function cssToObj(css?: string): CSSProperties {
  if (!css) return {}
  const cached = cache.get(css)
  if (cached) return cached
  const obj: Record<string, string> = {}
  for (const decl of css.split(';')) {
    const i = decl.indexOf(':')
    if (i === -1) continue
    const rawKey = decl.slice(0, i).trim()
    const value = decl.slice(i + 1).trim()
    if (!rawKey || !value) continue
    // Preserve custom properties (--x) as-is; camelCase the rest.
    const key = rawKey.startsWith('--')
      ? rawKey
      : rawKey.replace(/-([a-z])/g, (_, c: string) => c.toUpperCase())
    obj[key] = value
  }
  const result = obj as CSSProperties
  cache.set(css, result)
  return result
}

type Tag = keyof JSX.IntrinsicElements

type ElProps = {
  as?: Tag
  /** base css string */
  s?: string
  /** css applied while hovered */
  hover?: string
  /** css applied while pressed */
  active?: string
  /** css applied while focused */
  focus?: string
  children?: ReactNode
} & Record<string, unknown>

/**
 * A styled element that reproduces the design DSL's style / style-hover /
 * style-active / style-focus attributes using local interaction state.
 */
export function El({ as = 'div', s, hover, active, focus, children, ...rest }: ElProps) {
  const [isHover, setHover] = useState(false)
  const [isActive, setActive] = useState(false)
  const [isFocus, setFocus] = useState(false)

  const style = useMemo(() => {
    let css = s ?? ''
    if (isHover && hover) css += ';' + hover
    if (isFocus && focus) css += ';' + focus
    if (isActive && active) css += ';' + active
    return cssToObj(css)
  }, [s, hover, active, focus, isHover, isActive, isFocus])

  const handlers: Record<string, unknown> = {}
  if (hover) {
    const onEnter = rest.onMouseEnter as ((e: unknown) => void) | undefined
    const onLeave = rest.onMouseLeave as ((e: unknown) => void) | undefined
    handlers.onMouseEnter = (e: unknown) => {
      setHover(true)
      onEnter?.(e)
    }
    handlers.onMouseLeave = (e: unknown) => {
      setHover(false)
      setActive(false)
      onLeave?.(e)
    }
  }
  if (active) {
    handlers.onMouseDown = () => setActive(true)
    handlers.onMouseUp = () => setActive(false)
  }
  if (focus) {
    const onFocus = rest.onFocus as ((e: unknown) => void) | undefined
    const onBlur = rest.onBlur as ((e: unknown) => void) | undefined
    handlers.onFocus = (e: unknown) => {
      setFocus(true)
      onFocus?.(e)
    }
    handlers.onBlur = (e: unknown) => {
      setFocus(false)
      onBlur?.(e)
    }
  }

  return createElement(as, { ...rest, ...handlers, style }, children)
}
