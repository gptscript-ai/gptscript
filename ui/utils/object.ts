// Portions Copyright (c) 2014-2021 Rancher Labs, Inc. https://github.com/rancher/dashboard

// import { JSONPath } from 'jsonpath-plus';
import cloneDeep from 'lodash-es/cloneDeep.js'
import difference from 'lodash-es/difference.js'
import flattenDeep from 'lodash-es/flattenDeep.js'
import compact from 'lodash-es/compact.js'
import transform from 'lodash-es/transform.js'
import isEqual from 'lodash-es/isEqual.js'
import isObject from 'lodash-es/isObject.js'
import isEmpty from 'lodash-es/isEmpty.js'
import { addObject, isArray } from './array'
import { appendObjectPath, isNumeric, joinObjectPath, splitObjectPath } from './string'

export type JsonValue = null | undefined | string | number | boolean | JsonArray | JsonDict | object
export type JsonArray = JsonValue[]
export interface JsonDict extends Record<string, JsonValue> { }

type Change = {
  path: string
  op: string
  from: JsonValue
  value: JsonValue
}
interface ChangeSet extends Record<string, Change> {}

export function set(obj: JsonDict | null, path: string, value: any): any {
  let ptr: any = obj

  if (!ptr) {
    return
  }

  const parts = splitObjectPath(path)

  for (let i = 0; i < parts.length; i++) {
    const key = parts[i]

    if (i === parts.length - 1) {
      // Vue.set(ptr, key, value)
      ptr[key] = value
    } else if (!ptr[key]) {
      // Make sure parent keys exist
      // Vue.set(ptr, key, {})
      if ( isNumeric(parts[i + 1]) ) {
        ptr[key] = []
      } else {
        ptr[key] = {}
      }
    }

    ptr = ptr[key]
  }

  return obj
}

export function get(obj: JsonDict | object | null, path: string): any {
  /*
  if (path.startsWith('$')) {
    try {
      return JSONPath({
        path,
        json: obj,
        wrap: false,
      })
    } catch (e) {
      console.error('JSON Path error', e, path, obj) // eslint-disable-line no-console

      return '(JSON Path err)'
    }
  }
  */

  if ( !obj ) {
    throw new Error('Missing object')
  }

  if (!path.includes('.')) {
    return (obj as any)[path]
  }

  const parts = splitObjectPath(path)
  let ptr = obj as JsonValue

  for (let i = 0; i < parts.length; i++) {
    if (!ptr) {
      return
    }

    ptr = (ptr as any)[parts[i]]
  }

  return ptr
}

export function getter(path: string) {
  return function (obj: JsonDict) {
    return get(obj, path)
  }
}

export function remove(obj: JsonDict, path: string): JsonDict {
  const parentAry = splitObjectPath(path)
  const leafKey = parentAry.pop() as string
  let parent

  if ( parentAry.length ) {
    parent = get(obj, joinObjectPath(parentAry))
  } else {
    parent = obj
  }

  if (parent) {
    // Vue.set(parent, leafKey, undefined)
    parent[leafKey] = undefined
    delete parent[leafKey]
  }

  return obj
}

export function clone<T>(obj: T): T {
  return cloneDeep(obj)
}

export function definedKeys(obj: JsonDict): string[] {
  const keys = Object.keys(obj).map((key) => {
    const val = obj[key]

    if ( Array.isArray(val) ) {
      return key
    } else if ( isObject(val) ) {
      return definedKeys(val).map(subkey => appendObjectPath(key, subkey))
    } else {
      return key
    }
  })

  return compact(flattenDeep(keys))
}
