import type { Config } from 'tailwindcss'
import defaultTheme from 'tailwindcss/defaultTheme'


// import colors from 'tailwindcss/colors'

export default <Partial<Config>> {
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // 'gpt': {
        //   '50': '#f6f6f6',
        //   '100': '#e7e7e7',
        //   '200': '#d1d1d1',
        //   '300': '#b0b0b0',
        //   '400': '#888888',
        //   '500': '#6d6d6d',
        //   '600': '#5d5d5d',
        //   '700': '#4f4f4f',
        //   '800': '#454545',
        //   '900': '#1d1d1d',
        //   '950': '#080808',
        // },
      },

      fontFamily: {
        'sans': ['Poppins', ...defaultTheme.fontFamily.sans],
      },
    },
  },
}
