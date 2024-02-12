// Portions Copyright (c) 2014-2021 Rancher Labs, Inc. https://github.com/rancher/dashboard

type UriField = 'source' | 'protocol' | 'authority' | 'userInfo' | 'user' | 'password' | 'host' | 'port' | 'relative' | 'path' | 'directory' | 'file' | 'queryStr' | 'anchor'
type UriFields = {
  [key in UriField]: string; // eslint-disable-line no-unused-vars
}

type QueryParams = Record<string, null | undefined | string | string[]>

interface ParsedUri extends UriFields {
  query: QueryParams
}

export function partialEncodeUriComponent(v: string, spaceAsPlus = true, safeChars: string[]) {
  let escaped = encodeURIComponent(v)

  if ( spaceAsPlus ) {
    escaped = escaped.replace(/%20/g, '+')
  }

  if ( safeChars.length ) {
    for ( const c of safeChars ) {
      const re = new RegExp(encodeURIComponent(c), 'ig')

      escaped = escaped.replace(re, c)
    }
  }

  return escaped
}

export function addParam(url: string, key: string, val: string | string[] | null, spaceAsPlus = true, safeChars: string[] = []): string {
  let out = url + (url.includes('?') ? '&' : '?')

  // val can be a string or an array of strings
  if (!Array.isArray(val)) {
    val = [val as string]
  }

  out += val.map((v) => {
    if (v === null) {
      return `${ encodeURIComponent(key) }`
    } else {
      return `${ encodeURIComponent(key) }=${ partialEncodeUriComponent(v, spaceAsPlus, safeChars) }`
    }
  }).join('&')

  return out
}

export function addParams(url: string, params: QueryParams, spaceAsPlus = true, safeChars: string[] = []): string {
  if (params && typeof params === 'object') {
    Object.keys(params).forEach((key) => {
      url = addParam(url, key, params[key], spaceAsPlus, safeChars)
    })
  }

  return url
}

export function removeParam(url: string, key: string): string {
  const parsed = parseUrl(url)

  if (parsed.query?.[key]) {
    delete parsed.query[key]
  }

  return stringifyUrl(parsed)
}

export function parseLinkHeader(str: string) {
  const out: Record<string, string> = { }
  const lines = (str || '').split(',')

  for (const line of lines) {
    const match = line.match(/^\s*<([^>]+)>\s*;\s*rel\s*="(.*)"/)

    if (match) {
      out[match[2].toLowerCase()] = match[1]
    }
  }

  return out
}

export function isMaybeSecure(port: number, proto: string): boolean {
  const protocol = proto.toLowerCase()

  return portMatch([port], [443, 8443], ['443']) || protocol === 'https'
}

export function portMatch(ports: number[], equals: number[], endsWith: string[]): boolean {
  for (let i = 0; i < ports.length; i++) {
    const port = ports[i]

    if (equals.includes(port)) {
      return true
    }

    for (let j = 0; j < endsWith.length; j++) {
      const suffix = `${ endsWith[j] }`
      const portStr = `${ port }`

      if (portStr !== suffix && portStr.endsWith(suffix)) {
        return true
      }
    }
  }

  return false
}

// parseUri 1.2.2
// (c) Steven Levithan <stevenlevithan.com>
// https://javascriptsource.com/parseuri/
// MIT License
export function parseUrl(str: string): ParsedUri {
  const o = parseUrl.options
  const m = o.parser[o.strictMode ? 'strict' : 'loose'].exec(str)

  if (!m) {
    throw new Error(`Cannot parse as uri: ${ str }`)
  }

  const uri = {} as ParsedUri
  let i = 14

  while (i--) {
    uri[o.key[i]] = m[i] || ''
  }

  uri.query = {}
  uri.queryStr.replace(o.q.parser, (_, $1: string, $2: string): string => {
    if ($1) {
      uri[o.q.name][$1] = $2
    }

    return ''
  })

  return uri
}

parseUrl.options = {
  strictMode: false,
  key:        ['source', 'protocol', 'authority', 'userInfo', 'user', 'password', 'host', 'port', 'relative', 'path', 'directory', 'file', 'queryStr', 'anchor'],
  q:          {
    name:   'query',
    parser: /(?:^|&)([^&=]*)=?([^&]*)/g,
  },
  parser: {
    strict: /^(?:([^:\/?#]+):)?(?:\/\/((?:(([^:@]*)(?::([^:@]*))?)?@)?([^:\/?#]*)(?::(\d*))?))?((((?:[^?#\/]*\/)*)([^?#]*))(?:\?([^#]*))?(?:#(.*))?)/,
    loose:  /^(?:(?![^:@]+:[^:@\/]*@)([^:\/?#.]+):)?(?:\/\/)?((?:(([^:@]*)(?::([^:@]*))?)?@)?([^:\/?#]*)(?::(\d*))?)(((\/(?:[^?#](?![^?#\/]*\.[^?#\/.]+(?:[?#]|$)))*\/?)?([^?#\/]*))(?:\?([^#]*))?(?:#(.*))?)/,
  },
} as {
  strictMode: boolean
  key: UriField[]
  q: {
    name: 'query'
    parser: RegExp
  }
  parser: {
    strict: RegExp
    loose: RegExp
  }
}

export function stringifyUrl(uri: ParsedUri): string {
  let out = `${ uri.protocol }://`

  if (uri.user && uri.password) {
    out += `${ uri.user }:${ uri.password }@`
  } else if (uri.user) {
    out += `${ uri.user }@`
  }

  out += uri.host

  if (uri.port) {
    out += `:${ uri.port }`
  }

  out += uri.path || '/'

  out = addParams(out, uri.query || {})

  if (uri.anchor) {
    out += `#${ uri.anchor }`
  }

  return out
}
