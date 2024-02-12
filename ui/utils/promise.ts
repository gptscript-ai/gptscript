export async function usleep(delay: number) {
  return new Promise((resolve) => {
    setTimeout(() => {
      resolve(true)
    }, delay)
  })
}
