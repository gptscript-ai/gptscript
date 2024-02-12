// Portions Copyright (c) 2014-2021 Rancher Labs, Inc. https://github.com/rancher/dashboard
// Portions Copyright (c) 2020 Yehuda Katz, Tom Dale and Ember.js contributors

import xor from 'lodash-es/xor.js'
import _isArray from 'lodash-es/isArray.js'
import { get } from './object'
import type { JsonDict, JsonValue } from './object'

export function isArray(ary: any): boolean {
  return _isArray(ary)
}

export function removeObject<T>(ary: T[], obj: T): T[] {
  const idx = ary.indexOf(obj)

  if (idx >= 0) {
    ary.splice(idx, 1)
  }

  return ary
}

export function removeObjects<T>(ary: T[], objs: T[]): T[] {
  let i
  let indexes = []

  for (i = 0; i < objs.length; i++) {
    let idx = ary.indexOf(objs[i])

    // Find multiple copies of the same value
    while (idx !== -1) {
      indexes.push(idx)
      idx = ary.indexOf(objs[i], idx + 1)
    }
  }

  if (!indexes.length) {
    // That was easy...
    return ary
  }

  indexes = indexes.sort((a, b) => a - b)

  const ranges = []
  let first: number, last: number

  // Group all the indexes into contiguous ranges
  while (indexes.length) {
    first = <number>indexes.shift()
    last = first

    while (indexes.length && indexes[0] === last + 1) {
      last = <number>indexes.shift()
    }

    ranges.push({ start: first, end: last })
  }

  // Remove the items by range
  for (i = ranges.length - 1; i >= 0; i--) {
    const { start, end } = ranges[i]

    ary.splice(start, end - start + 1)
  }

  return ary
}

export function addObject<T>(ary: T[], obj: T): void {
  const idx = ary.indexOf(obj)

  if (idx === -1) {
    ary.push(obj)
  }
}

export function addObjects<T>(ary: T[], objs: T[]): void {
  const unique: any[] = []

  for (const obj of objs) {
    if (!ary.includes(obj) && !unique.includes(obj)) {
      unique.push(obj)
    }
  }

  ary.push(...unique)
}

export function insertAt<T>(ary: T[], idx: number, ...objs: T[]): void {
  ary.splice(idx, 0, ...objs)
}

export function removeAt<T>(ary: T[], idx: number, len = 1): void {
  if (idx < 0) {
    throw new Error('Index too low')
  }

  if (idx + len > ary.length) {
    throw new Error('Index + length too high')
  }

  ary.splice(idx, len)

  return ary
}

export function clear<T>(ary: T[]) {
  ary.splice(0, ary.length)
}

export function replaceWith<T>(ary: T[], ...objs: T[]) {
  ary.splice(0, ary.length, ...objs)
}

function findOrFilterBy(method: 'find' | 'filter', ary: null | any[], keyOrObj: string | JsonDict, val?: JsonValue): any {
  ary = ary || []

  if (typeof keyOrObj === 'object') {
    return ary[method]((item) => {
      for (const path in keyOrObj) {
        const want = keyOrObj[path]
        const have = get(item, path)

        if (typeof want === 'undefined') {
          if (!have) {
            return false
          }
        } else if (have !== want) {
          return false
        }
      }

      return true
    })
  } else if (val === undefined) {
    return ary[method](item => !!get(item, keyOrObj))
  } else {
    return ary[method](item => get(item, keyOrObj) === val)
  }
}

export function filterBy(ary: null | any[], keyOrObj: string | JsonDict, val?: JsonValue): any[] {
  return findOrFilterBy('filter', ary, keyOrObj, val)
}

export function findBy(ary: null | any[], keyOrObj: string | JsonDict, val?: JsonValue): any {
  return findOrFilterBy('find', ary, keyOrObj, val)
}

export function sameContents(a: any[], b: any[]) {
  return xor(a, b).length === 0
}

export function uniq(ary: any[]) {
  const out: any[] = []

  addObjects(out, ary)

  return out
}

export function toArray<T>(arg: T | T[]): T[] {
  if ( isArray(arg) ) {
    return arg as T[]
  } else {
    return [arg as T]
  }
}

export function fromArray<T>(arg: T | T[] | undefined, def?: T): T {
  if ( isArray(arg) ) {
    return (arg as T[])[0] || (def as T)
  } else {
    return (arg as T) || (def as T)
  }
}
