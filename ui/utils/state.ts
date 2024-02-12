export function iconForState(state: string) {
  switch (state) {
    case 'creating':
      return 'i-heroicons-clock'
    case 'running':
      return 'i-heroicons-cog animate-spin'
    case 'finished':
      return 'i-heroicons-beaker'
    case 'error':
    default:
      return 'i-heroicons-exclamation-triangle'
  }
}

export function colorForState(state: string, prefix='', brightness: number|boolean = false) {
  let out = prefix ? `${prefix}-` : ''

  switch (state) {
    case 'creating':
    case 'running':
      out += 'blue'
      break
    case 'finished':
      out += 'green'
      break
    case 'error':
    default:
      out += 'red'
      break
  }

  if ( brightness === true ) {
    out += '-500'
  } else if ( brightness ) {
    out += `${brightness}`
  }

  return out
}
