// Portions Copyright (c) 2014-2021 Rancher Labs, Inc. https://github.com/rancher/dashboard
import crypto from 'node:crypto'
import escapeRegExp from 'lodash-es/escapeRegExp.js'
import isNaN from 'lodash-es/isNaN.js'

const camelizeRegex = /[ _-]/
const decamelizeRegex = /([a-z\d])([A-Z])/g
const dasherizeRegex = /[ _]/g
const quotedMatch = /[^."']+|"([^"]*)"|'([^']*)'/g

const entityMap: Record<string, string> = {
  '&':  '&amp;',
  '<':  '&lt;',
  '>':  '&gt;',
  '"':  '&quot;',
  '\'': '&#39;',
  '/':  '&#47;',
}

const alpha = 'abcdefghijklmnopqrstuvwxyz'
const num = '0123456789'
const sym = '!@#$%^&*()_+-=[]{};:,./<>?|'

export const Charsets = {
  NUMERIC:         num,
  NO_VOWELS:       'bcdfghjklmnpqrstvwxz2456789',
  ALPHA:           alpha + alpha.toUpperCase(),
  ALPHA_NUM:       alpha + alpha.toUpperCase() + num,
  ALPHA_NUM_LOWER: alpha + num,
  ALPHA_LOWER:     alpha,
  ALPHA_UPPER:     alpha.toUpperCase(),
  HEX:             `${ num }ABCDEF`,
  PASSWORD:        alpha + alpha.toUpperCase() + num + alpha + alpha.toUpperCase() + num + sym,
  // ^-- includes alpha / ALPHA / num twice to reduce the occurrence of symbols
}

export function appendObjectPath(path: string, key: string) {
  let out: string

  if ( key.includes('.') ) {
    out = `${ path }."${ key }"`
  } else {
    out = `${ path }.${ key }`
  }

  if ( out.startsWith('.') ) {
    out = out.substring(1)
  }

  return out
}

export function asciiLike(str: string): boolean {
  if ( str.match(/[^\r\n\t\x20-\x7F]/) ) {
    return false
  }

  return true
}

export function titleCase(str: string): string {
  return dasherize(str).split('-').map(str => ucFirst(str)).join(' ')
}

export function toSnakeCase(str: string): string {
  return dasherize(str).split('-').map((str) => {
    return str.toUpperCase()
  }).join('_')
}

export function coerceToType(val: string, type: 'float' | 'int' | 'boolean'): number | boolean | null {
  if ( type === 'float' ) {
    // Coerce strings to floats
    return Number.parseFloat(val) || null // NaN becomes null
  } else if ( type === 'int' ) {
    // Coerce strings to ints
    const out = Number.parseInt(val, 10)

    if ( isNaN(out) ) {
      return null
    }

    return out
  } else if ( type === 'boolean') {
    // Coerce strings to boolean
    if (val.toLowerCase() === 'true') {
      return true
    } else if (val.toLowerCase() === 'false') {
      return false
    }
  }

  return null
}

export function camelize(str: string): string {
  return str.split(camelizeRegex).map((x, i) => i === 0 ? lcFirst(x) : ucFirst(x)).join('')
}

export function dasherize(str: string): string {
  return decamelize(str).replace(dasherizeRegex, '-')
}

export function decamelize(str: string): string {
  return str.replace(decamelizeRegex, '$1_$2').toLowerCase()
}

export function ensureRegex(strOrRegex: string | RegExp, exact = true, insensitive = true): RegExp {
  if ( typeof strOrRegex === 'string' ) {
    if ( exact ) {
      return new RegExp(`^${ escapeRegex(strOrRegex) }$`, insensitive ? 'i' : '')
    } else {
      return new RegExp(`${ escapeRegex(strOrRegex) }`, insensitive ? 'i' : '')
    }
  }

  return strOrRegex
}

export function searchToRegex(str: string): RegExp {
  const match = str.match(/^\/(\^)?(.*?)(\$)?\/([a-zA-Z]+)?$/)
  const hasUpper = !!str.match(/[A-Z]/)

  if ( match ) {
    //          [maybe ^      ]  [before]   [actual match]  [after]  [maybe $]
    const re = `${ match[1] || '' }(.*?)(${ match[2] || '' })(.*?)${ match[3] || '' }`

    return new RegExp(re, match[4] || '')
  }

  // (before)(match)(after)
  return new RegExp(`^(.*)(${ escapeRegExp(str) })(.*)$`, hasUpper ? '' : 'i')
}

export function escapeHtml(html: string): string {
  return String(html).replace(/[&<>"'\/]/g, (s) => {
    return entityMap[s]
  })
}

export function escapeRegex(string: string): string {
  return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') // $& means the whole matched string
}

export function formatPercent(value: number, maxPrecision = 2): string {
  if (value < 1 && maxPrecision >= 2) {
    return `${ Math.round(value * 100) / 100 }%`
  } else if (value < 10 && maxPrecision >= 1) {
    return `${ Math.round(value * 10) / 10 }%`
  } else {
    return `${ Math.round(value) }%`
  }
}

export function indent(lines: string | string[], count = 2, token = ' ', afterRegex?: RegExp): string {
  if (typeof lines === 'string') {
    lines = lines.split(/\n/)
  }

  const padStr = (Array.from({ length: count + 1 })).join(token)

  const out = lines.map((line: string) => {
    let prefix = ''
    let suffix = line

    if (afterRegex) {
      const match = line.match(afterRegex)

      if (match) {
        prefix = match[match.length - 1]
        suffix = line.substring(match[0].length)
      }
    }

    return `${ prefix }${ padStr }${ suffix }`
  })

  const str = out.join('\n')

  return str
}

export function isIpv4(ip: string): boolean {
  const reg = /^((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$/

  return reg.test(ip)
}

export function joinObjectPath(ary: string[]): string {
  let out = ''

  for ( const p of ary ) {
    out = appendObjectPath(out, p)
  }

  return out
}

export function lcFirst(str: string): string {
  return str.substring(0, 1).toLowerCase() + str.substring(1)
}

export function matchesSomeRegex(str: string, regexes: (string | RegExp)[] = []): boolean {
  return regexes.some((regexRaw: string | RegExp) => {
    const regex = ensureRegex(regexRaw)

    return str.match(regex)
  })
}

export function nlToBr(value: string): string {
  return escapeHtml(value).replace(/(\r\n|\r|\n)/g, '<br/>\n')
}

export function pluralize(str: string): string {
  if ( str.match(/.*[^aeiou]y$/i) ) {
    return `${ str.substring(0, str.length - 1) }ies`
  } else if ( str.endsWith('s') ) {
    return `${ str }es`
  } else {
    return `${ str }s`
  }
}

export function random32s(count: number): number[] {
  if ( count <= 0 ) {
    throw new Error("Can't generate a negative number of numbers")
  }

  if ( Math.floor(count) !== count ) {
    throw new Error('Count should be an integer')
  }

  const out = []
  let i: number

  if ( process.server || typeof window === 'undefined' ) {
    for ( i = 0; i < count; i++ ) {
      out[i] = crypto.randomBytes(4).readUInt32BE(0)
    }
  } else if (window.crypto && window.crypto.getRandomValues) {
    const tmp = new Uint32Array(count)

    window.crypto.getRandomValues(tmp)
    for (i = 0; i < tmp.length; i++) {
      out[i] = Math.round(tmp[i])
    }
  } else {
    for (i = 0; i < count; i++) {
      out[i] = Math.round(Math.random() * 4294967296) // Math.pow(2,32);
    }
  }

  return out
}

export function random32(): number {
  return random32s(1)[0]
}

export function randomStr(length = 16, chars = Charsets.ALPHA_NUM): string {
  if (!chars.length) {
    throw new Error('Charset is empty')
  }

  return random32s(length).map((val) => {
    return chars[val % chars.length]
  }).join('')
}

export function repeat(str: string, count: number) {
  return Array(count + 1).join(str)
}

export function splitLimit(str: string, separator: string, limit: number): string[] {
  const split = str.split(separator)

  if ( split.length < limit ) {
    return split
  }

  return [...split.slice(0, limit - 1), split.slice(limit - 1).join(separator)]
}

export function splitObjectPath(path: string): string[] {
  if ( path.includes('"') || path.includes('\'') ) {
    // Path with quoted section
    const matches = path.match(quotedMatch)

    if ( matches?.length ) {
      return matches.map(x => x.replace(/['"]/g, ''))
    }
  }

  // Regular path
  return path.split('.')
}

export function strPad(str: string, toLength: number, padChars = ' ', right = false): string {
  if (str.length >= toLength) {
    return str
  }

  const neededLen = toLength - str.length + 1
  const padStr = (Array.from({ length: neededLen })).join(padChars).substring(0, neededLen)

  if (right) {
    return str + padStr
  } else {
    return padStr + str
  }
}

export function ucFirst(str: string): string {
  return str.substring(0, 1).toUpperCase() + str.substring(1)
}

export function trim(str: string, direction: 'left' | 'right' | 'both' = 'both', chars = ' \n\r') {
  if ( direction === 'left' || direction === 'both' ) {
    const re = new RegExp(`^[${ escapeRegExp(chars) }]+`, 'g')

    str = str.replace(re, '')
  }

  if ( direction === 'right' || direction === 'both' ) {
    const re = new RegExp(`[${ escapeRegExp(chars) }]+$`, 'g')

    str = str.replace(re, '')
  }

  return str
}

export function singularize(str: string) {
  if ( str.endsWith('ies') ) {
    return str.replace(/ies$/, 'y')
  } else if ( str.endsWith('s') ) {
    return str.replace(/s$/, '')
  } else {
    return str
  }
}

export async function safeConcatName(...parts: string[]) {
  return safeConcatNameWithOptions(63, '-', ...parts)
}

export function depthOf(str: string, separator = '.') {
  let count = 0
  let i = -1

  do {
    i = str.indexOf(separator, i + 1)

    if ( i >= 0 ) {
      count++
    }
  } while ( i >= 0 )

  return count
}

export function nthLevelOf(str: string, level = 0, separator = '.') {
  if ( level === 0 ) {
    const idx = str.indexOf(separator)

    if ( idx === -1 ) {
      return str
    } else {
      return str.substring(0, idx)
    }
  } else if ( level === -1 ) {
    const parts = str.split(separator)

    parts.pop()

    return parts.join(separator)
  } else {
    return str.split(separator).slice(0, level + 1).join(separator)
  }
}

export function baseOf(str: string, separator = '.') {
  return nthLevelOf(str, -1, separator)
}

export function leafOf(str: string, separator = '.') {
  return str.split(separator).pop() as string
}

export function isChildOf(str: string, parent: string, separator = '.') {
  return str.startsWith(parent + separator)
}

export function isNumeric(str: string | number) {
  return (`${ str }`).match(/^([0-9]+\.)?[0-9]*$/)
}
