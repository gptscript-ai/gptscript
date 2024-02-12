import dayjs from 'dayjs'
import utc from 'dayjs/plugin/utc'
import timezone from 'dayjs/plugin/timezone'
import isToday from 'dayjs/plugin/isToday'

export default defineNuxtPlugin(() => {
  dayjs.extend(utc)
  dayjs.extend(timezone)
  dayjs.extend(isToday)
})
