<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="homeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      data-testid="home-iframe-override"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode -->
    <div
      v-else
      data-testid="home-html-override"
      v-html="homeContent"
    ></div>
  </div>

  <!-- Default Home Page -->
  <div
    v-else
    class="relative min-h-screen bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-gray-200 blueprint-bg"
  >
    <!-- Sticky Nav -->
    <header
      data-testid="home-nav"
      class="sticky top-0 z-40 border-b border-blue-100/50 bg-gray-50/80 backdrop-blur dark:border-dark-800/60 dark:bg-dark-950/80"
    >
      <nav class="mx-auto flex max-w-7xl items-center justify-between px-4 py-3 sm:px-6 lg:px-8">
        <!-- Logo -->
        <div class="flex items-center gap-2.5">
          <div class="flex h-9 w-9 items-center justify-center overflow-hidden rounded-xl bg-white ring-1 ring-primary-500/40 dark:bg-dark-900">
            <img :src="siteLogo || '/logo.svg'" alt="" class="h-full w-full object-contain" />
          </div>
          <span class="text-sm font-semibold tracking-tight">{{ siteName }}</span>
        </div>

        <!-- Nav Actions -->
        <div class="flex items-center gap-2">
          <!-- Docs -->
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="cursor-pointer rounded-lg p-2.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-dark-800 dark:hover:text-gray-200"
            :aria-label="t('homeV2.navDocs')"
          >
            <Icon name="book" size="sm" />
          </a>

          <!-- Locale Switcher -->
          <span data-testid="home-nav-locale">
            <LocaleSwitcher />
          </span>

          <!-- Theme Toggle -->
          <button
            data-testid="home-nav-theme"
            class="cursor-pointer rounded-lg p-2.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 dark:text-gray-400 dark:hover:bg-dark-800 dark:hover:text-gray-200"
            :aria-label="t(isDark ? 'homeV2.themeToLight' : 'homeV2.themeToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="sm" />
            <Icon v-else name="moon" size="sm" />
          </button>

          <!-- Auth Pill -->
          <router-link
            data-testid="home-nav-auth"
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex cursor-pointer items-center gap-2 rounded-full px-4 py-2 text-sm font-medium text-white transition-all hover:-translate-y-0.5 hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 bg-blue-600 dark:bg-blue-500"
          >
            <template v-if="isAuthenticated">
              <span class="flex h-6 w-6 items-center justify-center rounded-full bg-white/20 text-xs font-bold">
                {{ userInitial }}
              </span>
              <span>{{ t('homeV2.navDashboard') }}</span>
            </template>
            <template v-else>
              <span>{{ t('homeV2.navLogin') }}</span>
            </template>
          </router-link>
        </div>
      </nav>
    </header>

    <!-- Hero -->
    <main data-testid="home-hero" class="relative z-10 mx-auto max-w-7xl px-4 pt-12 sm:px-6 lg:px-8 lg:pt-20">
      <div class="grid gap-12 lg:grid-cols-2 lg:gap-16">
        <!-- Left Column -->
        <div class="text-center lg:text-left">
          <!-- Eyebrow -->
          <p class="font-mono text-xs uppercase tracking-[0.25em] text-blue-600 dark:text-blue-400 fade-slide-up-1">
            <span class="mr-2 inline-block h-1.5 w-1.5 rounded-full bg-teal-500"></span>{{ t('homeV2.eyebrow') }}
          </p>

          <!-- Title -->
          <h1 class="mt-4 font-serif text-4xl font-bold tracking-tight text-gray-900 dark:text-white sm:text-5xl lg:text-6xl xl:text-7xl fade-slide-up-2">
            {{ t('homeV2.title') }}
          </h1>

          <!-- Subtitle -->
          <p class="mt-4 max-w-xl text-lg leading-relaxed text-gray-600 dark:text-gray-400 fade-slide-up-3">
            {{ t('homeV2.subtitle') }}
          </p>

          <!-- CTA Row -->
          <div class="mt-8 flex flex-wrap items-center justify-center gap-4 lg:justify-start fade-slide-up-4">
            <router-link
              data-testid="home-cta-start"
              :to="isAuthenticated ? dashboardPath : '/login'"
              class="inline-flex cursor-pointer items-center gap-2 rounded-full bg-blue-600 px-6 py-3 text-sm font-semibold text-white shadow-sm transition-all hover:-translate-y-0.5 hover:bg-blue-700 hover:shadow-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 dark:bg-blue-500 dark:hover:bg-blue-400"
            >
              {{ isAuthenticated ? t('homeV2.ctaDashboard') : t('homeV2.ctaStart') }}
              <Icon name="arrowRight" size="sm" :stroke-width="2" />
            </router-link>

            <a
              v-if="docUrl"
              data-testid="home-cta-docs"
              :href="docUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="inline-flex cursor-pointer items-center gap-2 rounded-full border border-gray-300 px-5 py-2.5 text-sm font-medium text-gray-700 transition-all hover:-translate-y-0.5 hover:border-blue-300 hover:shadow-md dark:border-dark-700 dark:text-gray-300 dark:hover:border-blue-600"
            >
              {{ t('homeV2.ctaDocs') }}
              <Icon name="externalLink" size="sm" />
            </a>
          </div>
        </div>

        <!-- Right Column: Routing Diagram -->
        <div class="flex items-center justify-center lg:justify-end">
          <div data-testid="home-routing-diagram" class="w-full max-w-lg">
            <!-- Full diagram (desktop) -->
            <div data-testid="home-routing-diagram-desktop" class="hidden lg:block">
            <svg viewBox="0 0 560 400" fill="none" class="h-auto w-full svg-diagram">
              <defs>
                <filter id="glow-aqua" x="-20%" y="-20%" width="140%" height="140%">
                  <feGaussianBlur stdDeviation="4" result="blur" />
                  <feMerge><feMergeNode in="blur" /><feMergeNode in="SourceGraphic" /></feMerge>
                </filter>
              </defs>

              <!-- Base paths -->
              <path class="bp" d="M 140 200 C 210 200, 230 200, 255 200" />
              <path class="bp" d="M 305 200 C 370 80, 400 80, 430 80" />
              <path class="bp" d="M 305 200 C 370 140, 400 140, 430 140" />
              <path class="bp" d="M 305 200 C 370 200, 400 200, 430 200" />
              <path class="bp" d="M 305 200 C 370 260, 400 260, 430 260" />
              <path class="bp" d="M 305 200 C 370 320, 400 320, 430 320" />

              <!-- Pulse paths with glow (dark) -->
              <g filter="url(#glow-aqua)" class="pp-dark">
                <path class="pp pp-0" d="M 140 200 C 210 200, 230 200, 255 200" />
                <path class="pp pp-1" d="M 305 200 C 370 80, 400 80, 430 80" />
                <path class="pp pp-2" d="M 305 200 C 370 140, 400 140, 430 140" />
                <path class="pp pp-3" d="M 305 200 C 370 200, 400 200, 430 200" />
                <path class="pp pp-4" d="M 305 200 C 370 260, 400 260, 430 260" />
                <path class="pp pp-5" d="M 305 200 C 370 320, 400 320, 430 320" />
              </g>
              <!-- Pulse paths (light) -->
              <g class="pp-light">
                <path class="pp pp-0" d="M 140 200 C 210 200, 230 200, 255 200" />
                <path class="pp pp-1" d="M 305 200 C 370 80, 400 80, 430 80" />
                <path class="pp pp-2" d="M 305 200 C 370 140, 400 140, 430 140" />
                <path class="pp pp-3" d="M 305 200 C 370 200, 400 200, 430 200" />
                <path class="pp pp-4" d="M 305 200 C 370 260, 400 260, 430 260" />
                <path class="pp pp-5" d="M 305 200 C 370 320, 400 320, 430 320" />
              </g>

              <!-- Key Card (Left) -->
              <rect class="nc" x="20" y="170" width="120" height="60" rx="12" />
              <text x="80" y="198" text-anchor="middle" class="svg-label">API KEY</text>
              <text x="80" y="216" text-anchor="middle" class="svg-label svg-label-value">{{ t('homeV2.diagramLabelKey') }}</text>

              <!-- Gateway Hexagon (Center) -->
              <polygon class="gw" points="280,155 320,178 320,222 280,245 240,222 240,178" />
              <text x="280" y="204" text-anchor="middle" class="svg-label svg-label-gw">{{ t('homeV2.diagramLabelGateway') }}</text>

              <!-- Halo Ring -->
              <polygon class="d-halo" points="280,145 330,173 330,227 280,255 230,227 230,173" />

              <!-- Providers Label -->
              <text x="490" y="30" text-anchor="middle" class="svg-label svg-label-providers">{{ t('homeV2.diagramLabelProviders').toUpperCase() }}</text>

              <!-- Provider Chips -->
              <!-- Claude (terracotta accent) -->
              <rect class="nc" x="430" y="55" width="120" height="40" rx="10" />
              <rect class="nc-accent" x="430" y="55" width="120" height="40" rx="10" />
              <text x="455" y="80" class="svg-mark svg-mark-primary">C</text>
              <text x="475" y="80" class="svg-label">Claude</text>

              <!-- GPT -->
              <rect class="nc" x="430" y="115" width="120" height="40" rx="10" />
              <text x="455" y="140" class="svg-mark svg-mark-green">G</text>
              <text x="475" y="140" class="svg-label">GPT</text>

              <!-- Gemini -->
              <rect class="nc" x="430" y="175" width="120" height="40" rx="10" />
              <text x="455" y="200" class="svg-mark svg-mark-denim">G</text>
              <text x="475" y="200" class="svg-label">Gemini</text>

              <!-- Antigravity -->
              <rect class="nc" x="430" y="235" width="120" height="40" rx="10" />
              <text x="455" y="260" class="svg-mark svg-mark-purple">A</text>
              <text x="475" y="260" class="svg-label">Antigravity</text>

              <!-- More -->
              <rect class="nc" x="430" y="295" width="120" height="40" rx="10" />
              <text x="455" y="320" class="svg-mark svg-mark-gray">+</text>
              <text x="475" y="314" class="svg-label">{{ t('homeV2.providerMore') }}</text>
              <rect class="nc-badge" x="500" y="318" width="40" height="13" rx="4" />
              <text x="520" y="328" text-anchor="middle" class="svg-label svg-label-soon">{{ t('homeV2.providerSoon') }}</text>
            </svg>
            </div>

            <!-- Simplified diagram (mobile) -->
            <div data-testid="home-routing-diagram-mobile" class="lg:hidden">
              <svg viewBox="0 0 340 88" fill="none" class="mx-auto h-auto w-full max-w-sm svg-diagram">
                <!-- Base paths -->
                <path class="bp" d="M 104 44 C 124 44, 132 44, 149 44" />
                <path class="bp" d="M 191 44 C 212 44, 216 22, 240 22" />
                <path class="bp" d="M 191 44 C 212 44, 216 44, 240 44" />
                <path class="bp" d="M 191 44 C 212 44, 216 66, 240 66" />
                <!-- Request pulses -->
                <path class="pp pp-0" d="M 104 44 C 124 44, 132 44, 149 44" />
                <path class="pp pp-3" d="M 191 44 C 212 44, 216 44, 240 44" />

                <!-- Key chip -->
                <rect class="nc" x="8" y="26" width="96" height="36" rx="10" />
                <text x="56" y="41" text-anchor="middle" class="svg-label svg-label-sm">API KEY</text>
                <text x="56" y="54" text-anchor="middle" class="svg-label svg-label-value svg-label-sm">{{ t('homeV2.diagramLabelKey') }}</text>

                <!-- Gateway hexagon -->
                <polygon class="gw" points="170,22 189,33 189,55 170,66 151,55 151,33" />
                <text x="170" y="48" text-anchor="middle" class="svg-label svg-label-gw svg-label-sm">{{ t('homeV2.diagramLabelGateway') }}</text>

                <!-- Provider mini chips -->
                <rect class="nc" x="240" y="12" width="92" height="20" rx="6" />
                <text x="252" y="26" class="svg-mark svg-mark-primary svg-mark-sm">C</text>
                <text x="264" y="26" class="svg-label svg-label-sm">Claude</text>

                <rect class="nc" x="240" y="34" width="92" height="20" rx="6" />
                <text x="252" y="48" class="svg-mark svg-mark-green svg-mark-sm">G</text>
                <text x="264" y="48" class="svg-label svg-label-sm">GPT</text>

                <rect class="nc" x="240" y="56" width="92" height="20" rx="6" />
                <text x="252" y="70" class="svg-mark svg-mark-gray svg-mark-sm">+</text>
                <text x="264" y="70" class="svg-label svg-label-sm">{{ t('homeV2.providerMore') }}</text>
              </svg>
            </div>

            <!-- Status line -->
            <p class="mt-3 text-center font-mono text-xs uppercase tracking-[0.2em] text-gray-400 dark:text-gray-500">
              <span class="d-blink mr-2 inline-block h-2 w-2 rounded-full bg-teal-500"></span>{{ t('homeV2.diagramStatus') }}
            </p>
          </div>
        </div>
      </div>
    </main>

    <!-- Feature Grid -->
    <section class="mx-auto mt-16 max-w-7xl px-4 sm:px-6 lg:px-8">
      <div class="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
        <!-- Gateway (spans 2 cols on lg) -->
        <div class="group rounded-2xl border border-blue-100/60 bg-white/70 p-6 backdrop-blur transition-all duration-200 hover:-translate-y-0.5 hover:border-blue-300 hover:shadow-lg dark:border-dark-800/60 dark:bg-dark-900/60 dark:hover:border-blue-700 lg:col-span-2">
          <div class="flex items-start gap-4">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-600 dark:bg-dark-800 dark:text-blue-400">
              <Icon name="server" size="md" />
            </div>
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('homeV2.featGatewayTitle') }}</h2>
              <p class="mt-1 text-sm leading-relaxed text-gray-600 dark:text-gray-400">{{ t('homeV2.featGatewayDesc') }}</p>
            </div>
          </div>
        </div>

        <!-- Scheduling -->
        <div class="group rounded-2xl border border-blue-100/60 bg-white/70 p-6 backdrop-blur transition-all duration-200 hover:-translate-y-0.5 hover:border-blue-300 hover:shadow-lg dark:border-dark-800/60 dark:bg-dark-900/60 dark:hover:border-blue-700">
          <div class="flex items-start gap-4">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-600 dark:bg-dark-800 dark:text-blue-400">
              <Icon name="cpu" size="md" />
            </div>
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('homeV2.featSchedulingTitle') }}</h2>
              <p class="mt-1 text-sm leading-relaxed text-gray-600 dark:text-gray-400">{{ t('homeV2.featSchedulingDesc') }}</p>
            </div>
          </div>
        </div>

        <!-- Billing -->
        <div class="group rounded-2xl border border-blue-100/60 bg-white/70 p-6 backdrop-blur transition-all duration-200 hover:-translate-y-0.5 hover:border-blue-300 hover:shadow-lg dark:border-dark-800/60 dark:bg-dark-900/60 dark:hover:border-blue-700">
          <div class="flex items-start gap-4">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-600 dark:bg-dark-800 dark:text-blue-400">
              <Icon name="chart" size="md" />
            </div>
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('homeV2.featBillingTitle') }}</h2>
              <p class="mt-1 text-sm leading-relaxed text-gray-600 dark:text-gray-400">{{ t('homeV2.featBillingDesc') }}</p>
            </div>
          </div>
        </div>

        <!-- Sticky Sessions -->
        <div class="group rounded-2xl border border-blue-100/60 bg-white/70 p-6 backdrop-blur transition-all duration-200 hover:-translate-y-0.5 hover:border-blue-300 hover:shadow-lg dark:border-dark-800/60 dark:bg-dark-900/60 dark:hover:border-blue-700 md:col-span-2 lg:col-span-1">
          <div class="flex items-start gap-4">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-600 dark:bg-dark-800 dark:text-blue-400">
              <Icon name="shield" size="md" />
            </div>
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('homeV2.featStickyTitle') }}</h2>
              <p class="mt-1 text-sm leading-relaxed text-gray-600 dark:text-gray-400">{{ t('homeV2.featStickyDesc') }}</p>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- Quickstart -->
    <section data-testid="home-quickstart" class="mx-auto mt-16 max-w-7xl px-4 sm:px-6 lg:px-8">
      <div class="mx-auto max-w-3xl text-center">
        <p class="font-mono text-xs uppercase tracking-[0.25em] text-blue-600 dark:text-blue-400">{{ t('homeV2.quickstartTitle') }}</p>
      </div>
      <div class="mx-auto mt-6 max-w-3xl overflow-hidden rounded-2xl bg-dark-950 text-gray-200 shadow-xl">
        <!-- Step 1 -->
        <div class="border-b border-dark-800 px-6 py-5">
          <span class="font-mono text-xs text-teal-400">1 / 3</span>
          <p class="mt-1 font-mono text-sm text-gray-300">{{ t('homeV2.quickstartStep1') }}</p>
        </div>
        <!-- Step 2 -->
        <div class="border-b border-dark-800 px-6 py-5">
          <span class="font-mono text-xs text-teal-400">2 / 3</span>
          <p class="mt-1 font-mono text-sm text-gray-300">{{ t('homeV2.quickstartStep2') }}</p>
          <div class="mt-3 overflow-hidden rounded-lg bg-dark-900 px-4 py-3 font-mono text-sm">
            <span class="text-gray-500"># Point your SDK</span><br/>
            <span class="text-teal-300">export</span> <span class="text-gray-300">ANTHROPIC_BASE_URL</span><span class="text-gray-500">=</span><span class="text-amber-300">"https://your-gateway.example.com"</span><br/>
            <span class="text-teal-300">export</span> <span class="text-gray-300">ANTHROPIC_API_KEY</span><span class="text-gray-500">=</span><span class="text-amber-300">"sk-your-gateway-key"</span>
          </div>
        </div>
        <!-- Step 3 -->
        <div class="px-6 py-5">
          <span class="font-mono text-xs text-teal-400">3 / 3</span>
          <p class="mt-1 font-mono text-sm text-gray-300">{{ t('homeV2.quickstartStep3') }}</p>
          <div class="mt-3 overflow-hidden rounded-lg bg-dark-900 px-4 py-3 font-mono text-sm">
            <span class="text-teal-300">curl</span> <span class="text-gray-300">-X POST</span> <span class="text-amber-300">"$ANTHROPIC_BASE_URL/v1/messages"</span> <span class="text-gray-500">\</span><br/>
            &nbsp;&nbsp;<span class="text-gray-300">-H</span> <span class="text-amber-300">"x-api-key: $ANTHROPIC_API_KEY"</span> <span class="text-gray-500">\</span><br/>
            &nbsp;&nbsp;<span class="text-gray-300">-H</span> <span class="text-amber-300">"content-type: application/json"</span> <span class="text-gray-500">\</span><br/>
            &nbsp;&nbsp;<span class="text-gray-300">-d</span> <span class="text-amber-300">'{"model": "claude-sonnet-4-5"}'</span>
          </div>
        </div>
      </div>
    </section>

    <!-- Providers -->
    <section data-testid="home-providers" class="mx-auto mt-16 max-w-7xl px-4 sm:px-6 lg:px-8">
      <div class="text-center">
        <h2 class="font-serif text-2xl font-bold text-gray-900 dark:text-white sm:text-3xl">{{ t('homeV2.providersTitle') }}</h2>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">{{ t('homeV2.providersDesc') }}</p>
      </div>
      <div class="mt-8 flex flex-wrap items-center justify-center gap-3">
        <div class="flex items-center gap-2.5 rounded-xl border border-blue-100/60 bg-white/60 px-5 py-3 backdrop-blur dark:border-dark-800/60 dark:bg-dark-900/60">
          <span class="flex h-7 w-7 items-center justify-center rounded-md bg-primary-500/15 text-xs font-bold text-primary-500">C</span>
          <span class="font-mono text-sm text-gray-700 dark:text-gray-200">Claude</span>
        </div>
        <div class="flex items-center gap-2.5 rounded-xl border border-blue-100/60 bg-white/60 px-5 py-3 backdrop-blur dark:border-dark-800/60 dark:bg-dark-900/60">
          <span class="flex h-7 w-7 items-center justify-center rounded-md bg-green-500/15 text-xs font-bold text-green-500">G</span>
          <span class="font-mono text-sm text-gray-700 dark:text-gray-200">GPT</span>
        </div>
        <div class="flex items-center gap-2.5 rounded-xl border border-blue-100/60 bg-white/60 px-5 py-3 backdrop-blur dark:border-dark-800/60 dark:bg-dark-900/60">
          <span class="flex h-7 w-7 items-center justify-center rounded-md bg-blue-500/15 text-xs font-bold text-blue-500 dark:text-blue-400">G</span>
          <span class="font-mono text-sm text-gray-700 dark:text-gray-200">Gemini</span>
        </div>
        <div class="flex items-center gap-2.5 rounded-xl border border-blue-100/60 bg-white/60 px-5 py-3 backdrop-blur dark:border-dark-800/60 dark:bg-dark-900/60">
          <span class="flex h-7 w-7 items-center justify-center rounded-md bg-purple-500/15 text-xs font-bold text-purple-500">A</span>
          <span class="font-mono text-sm text-gray-700 dark:text-gray-200">Antigravity</span>
        </div>
        <div class="flex items-center gap-2.5 rounded-xl border border-gray-200/60 bg-white/40 px-5 py-3 opacity-70 backdrop-blur dark:border-dark-800/60 dark:bg-dark-900/40">
          <span class="flex h-7 w-7 items-center justify-center rounded-md bg-gray-400/15 text-xs font-bold text-gray-400">+</span>
          <span class="font-mono text-sm text-gray-500 dark:text-gray-400">{{ t('homeV2.providerMore') }}</span>
          <span class="rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-400 dark:bg-dark-800 dark:text-gray-500">{{ t('homeV2.providerSoon') }}</span>
        </div>
      </div>
    </section>

    <!-- CTA Band -->
    <section data-testid="home-cta-band" class="mx-auto mt-16 max-w-7xl px-4 sm:px-6 lg:px-8">
      <div class="rounded-3xl bg-gradient-to-r from-blue-600 to-teal-600 px-8 py-12 text-center text-white sm:px-12 sm:py-16">
        <h2 class="font-serif text-2xl font-bold sm:text-3xl">{{ t('homeV2.ctaBandTitle') }}</h2>
        <p class="mx-auto mt-2 max-w-lg text-sm text-white/80 sm:text-base">{{ t('homeV2.ctaBandDesc') }}</p>
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="mt-6 inline-flex cursor-pointer items-center gap-2 rounded-full bg-white px-6 py-3 text-sm font-semibold text-blue-700 shadow-lg transition-all hover:-translate-y-0.5 hover:shadow-xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white"
        >
          {{ t('homeV2.ctaBandBtn') }}
          <Icon name="arrowRight" size="sm" :stroke-width="2" />
        </router-link>
      </div>
    </section>

    <!-- Footer -->
    <footer data-testid="home-footer" class="mt-16 border-t border-blue-100/40 dark:border-dark-800/40">
      <div class="mx-auto flex max-w-7xl flex-col items-center justify-between gap-4 px-4 py-8 sm:flex-row sm:px-6 lg:px-8">
        <p class="font-mono text-xs text-gray-400 dark:text-gray-500">
          &copy; {{ currentYear }} {{ siteName }} {{ t('homeV2.footerRights') }}
        </p>
        <div class="flex items-center gap-5">
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="font-mono text-xs text-gray-400 transition-colors hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300"
          >
            {{ t('homeV2.footerDocs') }}
          </a>
          <a
            :href="githubUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="font-mono text-xs text-gray-400 transition-colors hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300"
          >
            {{ t('homeV2.footerGithub') }}
          </a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { useTheme } from '@/composables/useTheme'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')

const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const { isDark, toggleTheme } = useTheme()
const githubUrl = 'https://github.com/Redstonexs/sub2api'

const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})
const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  authStore.checkAuth()
  if (!appStore.cachedPublicSettings) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
.blueprint-bg {
  background-image: radial-gradient(circle, rgba(55, 111, 192, 0.04) 1px, transparent 1px);
  background-size: 24px 24px;
}

:deep(.dark) .blueprint-bg {
  background-image: radial-gradient(circle, rgba(62, 180, 171, 0.06) 1px, transparent 1px);
}

/* SVG Diagram Styles */
.svg-diagram .bp {
  stroke: #93b5d6;
  stroke-width: 1.5;
  fill: none;
}

:deep(.dark) .svg-diagram .bp {
  stroke: #6b9ec7;
}

.svg-diagram .pp {
  stroke-dasharray: 6 300;
  stroke-width: 1.5;
  fill: none;
  animation: dash-pulse 2s linear infinite;
}

.svg-diagram .pp-0 { stroke: #3EB4AB; animation-delay: 0s; }
.svg-diagram .pp-1 { stroke: #3EB4AB; animation-delay: 0.4s; }
.svg-diagram .pp-2 { stroke: #3EB4AB; animation-delay: 0.8s; }
.svg-diagram .pp-3 { stroke: #3EB4AB; animation-delay: 1.2s; }
.svg-diagram .pp-4 { stroke: #3EB4AB; animation-delay: 1.6s; }
.svg-diagram .pp-5 { stroke: #3EB4AB; animation-delay: 2s; }

:deep(.dark) .svg-diagram .pp {
  stroke: #5ec4c0;
  filter: drop-shadow(0 0 6px rgba(62, 180, 171, 0.5));
}

.svg-diagram .pp-dark {
  display: none;
}

:deep(.dark) .svg-diagram .pp-dark {
  display: block;
}

:deep(.dark) .svg-diagram .pp-light {
  display: none;
}

.svg-diagram .nc {
  fill: #fff;
  stroke: #c5d8ec;
  stroke-width: 1;
}

:deep(.dark) .svg-diagram .nc {
  fill: #1c2333;
  stroke: #2d3748;
}

.svg-diagram .nc-accent {
  fill: none;
  stroke: #D0685B;
  stroke-width: 1.5;
  opacity: 0.6;
}

.svg-diagram .nc-badge {
  fill: #f3f4f6;
}

:deep(.dark) .svg-diagram .nc-badge {
  fill: #1f2937;
}

.svg-diagram .gw {
  fill: #eef4fb;
  stroke: #7ba3cc;
  stroke-width: 1.5;
}

:deep(.dark) .svg-diagram .gw {
  fill: #1a2636;
  stroke: #5b8bb5;
}

.svg-diagram .svg-label {
  font-family: ui-monospace, monospace;
  fill: #4b5563;
  font-size: 10px;
}

:deep(.dark) .svg-diagram .svg-label {
  fill: #9ca3af;
}

.svg-diagram .svg-label-value {
  fill: #1f2937;
  font-size: 11px;
  font-weight: 600;
}

:deep(.dark) .svg-diagram .svg-label-value {
  fill: #e5e7eb;
}

.svg-diagram .svg-label-gw {
  fill: #2563eb;
  font-size: 12px;
  font-weight: 700;
}

:deep(.dark) .svg-diagram .svg-label-gw {
  fill: #93b5d6;
}

.svg-diagram .svg-label-providers {
  font-size: 9px;
  letter-spacing: 2px;
  fill: #9ca3af;
}

:deep(.dark) .svg-diagram .svg-label-providers {
  fill: #6b7280;
}

.svg-diagram .svg-label-soon {
  font-size: 8px;
  fill: #9ca3af;
}

:deep(.dark) .svg-diagram .svg-label-soon {
  fill: #6b7280;
}

.svg-diagram .svg-mark {
  font-family: ui-monospace, monospace;
  font-size: 13px;
  font-weight: 700;
}

.svg-diagram .svg-label-sm { font-size: 8.5px; }
.svg-diagram .svg-mark-sm { font-size: 10px; }

.svg-diagram .svg-mark-primary { fill: #D0685B; }
.svg-diagram .svg-mark-green { fill: #22c55e; }
.svg-diagram .svg-mark-denim { fill: #376FC0; }
:deep(.dark) .svg-diagram .svg-mark-denim { fill: #6b9ec7; }
.svg-diagram .svg-mark-purple { fill: #a855f7; }
.svg-diagram .svg-mark-gray { fill: #9ca3af; }

@keyframes dash-pulse {
  from { stroke-dashoffset: 0; }
  to { stroke-dashoffset: -306; }
}

@keyframes halo-pulse {
  0%, 100% { transform: scale(1); opacity: 0.4; }
  50% { transform: scale(1.06); opacity: 0.15; }
}

@keyframes status-blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

@keyframes fade-slide-up {
  from { opacity: 0; transform: translateY(12px); }
  to { opacity: 1; transform: translateY(0); }
}

.svg-diagram .d-halo {
  stroke: #3EB4AB;
  stroke-width: 1;
  fill: none;
  opacity: 0.5;
  animation: halo-pulse 3s ease-in-out infinite;
  transform-origin: 280px 200px;
}

.d-blink {
  animation: status-blink 2s ease-in-out infinite;
}

.fade-slide-up-1 {
  animation: fade-slide-up 0.6s ease-out both;
  animation-delay: 0.1s;
}

.fade-slide-up-2 {
  animation: fade-slide-up 0.6s ease-out both;
  animation-delay: 0.2s;
}

.fade-slide-up-3 {
  animation: fade-slide-up 0.6s ease-out both;
  animation-delay: 0.3s;
}

.fade-slide-up-4 {
  animation: fade-slide-up 0.6s ease-out both;
  animation-delay: 0.4s;
}

@media (prefers-reduced-motion: reduce) {
  .svg-diagram .d-halo,
  .svg-diagram .pp,
  .d-blink,
  .fade-slide-up-1,
  .fade-slide-up-2,
  .fade-slide-up-3,
  .fade-slide-up-4 {
    animation: none !important;
    opacity: 1;
    transform: none;
  }
}
</style>
