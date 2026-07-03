/** @type {import('tailwindcss').Config} */

// ============================================================================
// 全局暖色调色板重映射 (Warm palette remap)
//
// 用暖调家族整体替换 Tailwind 内置色相家族（与下方 gray 覆盖同一机制），
// 让 ~3.6k 处硬编码的 blue/green/purple/... 类名无需改动即可换上品牌暖色。
//
// 生成配方（OKLCH，脚本推导，非手调）：
//   - 每档保留原生 Tailwind 的亮度 L（既有类名组合的对比度关系不变）
//   - 色相 H 旋转到家族锚点色
//   - 彩度 C 按锚点推导系数衰减（约 0.55–0.85；50–200 浅档系数略高，
//     避免在奶油底色上褪成白）
// 校验结果：white-on-600、700-on-100、暗色 400-on-900/30 徽章组合
// 均不低于被替换的原生家族。
//
// 注意：支付品牌色（#635bff / #00AEEF / #2BB741 / #14171A，写在组件里）
// 不属于本映射，禁止改动。
// ============================================================================

// Brick 砖红 —— 替换 red / rose（锚点 #C0564A）
const brick = {
  50: '#FCF3F1',
  100: '#FAE4E1',
  200: '#F6CEC8',
  300: '#E8B1A8',
  400: '#DD877B',
  500: '#D0685B',
  600: '#BE5448',
  700: '#9F443A',
  800: '#843931',
  900: '#6D322B',
  950: '#3B1713'
}

// Terracotta 陶土 —— 替换 orange（锚点 #D97757，近主色）
const terracotta = {
  50: '#FFF6F4',
  100: '#FFEBE5',
  200: '#FFD3C5',
  300: '#F7B8A4',
  400: '#EB987D',
  500: '#E28364',
  600: '#D06F4F',
  700: '#AB573C',
  800: '#884630',
  900: '#6D3A29',
  950: '#3A1C12'
}

// Ochre 赭金 —— 替换 amber / yellow（锚点 #D9A03C）
const ochre = {
  50: '#FFFAF3',
  100: '#FFF0DB',
  200: '#FFE1B5',
  300: '#FFCE82',
  400: '#F9BE5E',
  500: '#E2A846',
  600: '#BE8927',
  700: '#986901',
  800: '#7A5300',
  900: '#644504',
  950: '#382400'
}

// Fern 蕨绿 —— 替换 green / emerald / lime（锚点 #5E9B52）
const fern = {
  50: '#F4FCF2',
  100: '#E6F9E2',
  200: '#CEF2C7',
  300: '#B2E5A8',
  400: '#8ED381',
  500: '#73BA65',
  600: '#5D9A51',
  700: '#4A7941',
  800: '#3C5F35',
  900: '#324E2D',
  950: '#192B15'
}

// Aqua 海沫青 —— 替换 cyan / teal（锚点 #2E958D）
const aqua = {
  50: '#F1FDFB',
  100: '#CFFAF5',
  200: '#9FF4EB',
  300: '#73E6DC',
  400: '#50D0C6',
  500: '#3EB4AB',
  600: '#32918A',
  700: '#29746E',
  800: '#235C57',
  900: '#1F4D49',
  950: '#0E2E2B'
}

// Denim 牛仔蓝 —— 替换 blue / indigo / sky（锚点 #4E7FC6）
const denim = {
  50: '#F1F6FD',
  100: '#DFE9F9',
  200: '#C6DAF6',
  300: '#A8C3E9',
  400: '#7EA4DD',
  500: '#5688CF',
  600: '#376FC0',
  700: '#255EAE',
  800: '#204D8D',
  900: '#214271',
  950: '#172A45'
}

// Plum 灰紫 —— 替换 purple / violet / fuchsia / pink（锚点 #9A67A8）
const plum = {
  50: '#FAF5FC',
  100: '#F4E9F7',
  200: '#EBD7F1',
  300: '#D7BBDF',
  400: '#BF92CC',
  500: '#A96DB9',
  600: '#9656A7',
  700: '#814690',
  800: '#6B3C78',
  900: '#573161',
  950: '#3C1C44'
}

// 中性色 - Warm Stone 暖石色系（覆盖内置 gray，并接管 slate/zinc/neutral/stone）
const warmGray = {
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
}

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
        gray: warmGray,
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
        },

        // ---- 内置色相家族重映射（见文件头注释）----
        red: brick,
        rose: brick,
        orange: terracotta,
        amber: ochre,
        yellow: ochre,
        green: fern,
        emerald: fern,
        lime: fern,
        cyan: aqua,
        teal: aqua,
        blue: denim,
        indigo: denim,
        sky: denim,
        purple: plum,
        violet: plum,
        fuchsia: plum,
        pink: plum,
        slate: warmGray,
        zinc: warmGray,
        neutral: warmGray,
        stone: warmGray,

        // ---- 语义状态色（新代码优先使用这些别名）----
        success: fern,
        warning: ochre,
        danger: brick,
        info: denim
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
