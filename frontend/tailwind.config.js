/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // 主色调 - Clay/Terracotta 陶土色系 (Anthropic editorial)
        primary: {
          50: '#FBF3EF',
          100: '#F5E1D7',
          200: '#EBC3AF',
          300: '#DFA184',
          400: '#D98E6A',
          500: '#CC785C',
          600: '#B05C40',
          700: '#8F4A34',
          800: '#73402F',
          900: '#5E3728',
          950: '#331B14'
        },
        // 中性色覆盖 - Warm Stone 暖石色系 (覆盖 Tailwind 内置 gray)
        gray: {
          50: '#FAF9F5',
          100: '#F3F1EA',
          200: '#E7E3D8',
          300: '#D5CFC0',
          400: '#ABA493',
          500: '#827B6C',
          600: '#605A4E',
          700: '#45413A',
          800: '#2B2823',
          900: '#1A1815',
          950: '#100F0C'
        },
        // 辅助色 - 暖炭/暖褐 (镜像 dark)
        accent: {
          50: '#F8F6F0',
          100: '#EDEAE1',
          200: '#DAD5C9',
          300: '#BBB4A6',
          400: '#938C7E',
          500: '#6E685C',
          600: '#4C473E',
          700: '#39352E',
          800: '#2A2722',
          900: '#211F1A',
          950: '#1A1814'
        },
        // 深色模式背景 - Warm Charcoal 暖炭色系
        dark: {
          50: '#F8F6F0',
          100: '#EDEAE1',
          200: '#DAD5C9',
          300: '#BBB4A6',
          400: '#938C7E',
          500: '#6E685C',
          600: '#4C473E',
          700: '#39352E',
          800: '#2A2722',
          900: '#211F1A',
          950: '#1A1814'
        }
      },
      fontFamily: {
        // 衬线展示字体 - Fraunces (标题/展示)
        serif: ['Fraunces', 'Fraunces Variable', 'Georgia', 'Songti SC', 'SimSun', 'serif'],
        // 无衬线 UI 字体 - Inter 优先, 保留 CJK 系统回退
        sans: [
          'Inter',
          'Inter Variable',
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Helvetica Neue',
          'Arial',
          'PingFang SC',
          'Hiragino Sans GB',
          'Microsoft YaHei',
          'sans-serif'
        ],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace']
      },
      boxShadow: {
        // 扁平化的暖色阴影 (保留 token 名以免引用失效)
        glass: '0 1px 2px rgba(20, 15, 10, 0.04)',
        'glass-sm': '0 1px 2px rgba(20, 15, 10, 0.03)',
        glow: '0 1px 2px rgba(20, 15, 10, 0.04)',
        'glow-lg': '0 6px 20px rgba(20, 15, 10, 0.06)',
        card: '0 1px 2px rgba(20, 15, 10, 0.04)',
        'card-hover': '0 6px 20px rgba(20, 15, 10, 0.06)',
        'inner-glow': 'inset 0 1px 0 rgba(255, 255, 255, 0.08)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': 'linear-gradient(135deg, #CC785C 0%, #B05C40 100%)',
        'gradient-dark': 'linear-gradient(135deg, #2A2722 0%, #1A1814 100%)',
        'gradient-glass':
          'linear-gradient(135deg, rgba(255,255,255,0.06) 0%, rgba(255,255,255,0.02) 100%)',
        // 几乎无形的暖奶油色微光
        'mesh-gradient':
          'radial-gradient(at 40% 20%, rgba(204, 120, 92, 0.05) 0px, transparent 50%), radial-gradient(at 80% 0%, rgba(176, 92, 64, 0.03) 0px, transparent 50%), radial-gradient(at 0% 50%, rgba(204, 120, 92, 0.03) 0px, transparent 50%)'
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'glow 2s ease-in-out infinite alternate'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' }
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        },
        glow: {
          '0%': { boxShadow: '0 1px 2px rgba(20, 15, 10, 0.04)' },
          '100%': { boxShadow: '0 6px 20px rgba(20, 15, 10, 0.08)' }
        }
      },
      backdropBlur: {
        xs: '2px'
      },
      borderRadius: {
        '4xl': '2rem'
      }
    }
  },
  plugins: []
}
