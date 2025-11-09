# Frontend Modernization Plan - Go Monitoring Dashboard

## Overview
This document outlines a comprehensive plan to modernize the frontend architecture, introducing reactive frameworks and improving maintainability, performance, and developer experience while preserving the excellent design system and functionality.

---

## 🎯 Current State Analysis

### Architecture Overview
- **Framework**: Vanilla JavaScript with manual DOM manipulation
- **State Management**: Central state object with imperative updates
- **Styling**: Single 2,598-line CSS file with comprehensive design system
- **Templates**: Go templates (templ) with server-side rendering
- **Build Process**: No build system, direct file serving
- **JavaScript Size**: ~3,000 lines across 18 modules

### Key Issues Identified
1. **Manual DOM Manipulation**: 50+ instances of `document.getElementById()`
2. **Scattered Update Logic**: UI updates spread across multiple files
3. **No Reactivity**: State changes require manual DOM updates
4. **Monolithic CSS**: Single large CSS file, difficult to maintain
5. **Testing Challenges**: Direct DOM manipulation hard to test
6. **Code Duplication**: Repeated patterns for state updates and DOM queries

---

## 🚀 Modernization Strategy

### Phase 1: Foundation Improvements (Week 1-2)
**Goal**: Establish modern development practices without breaking changes

### Phase 2: Reactive Framework Integration (Week 3-4)
**Goal**: Introduce Alpine.js for declarative UI and reactive state management

### Phase 3: Component Architecture (Week 5-6)
**Goal**: Implement Web Components with Lit for reusable UI elements

### Phase 4: Advanced Features (Week 7-8)
**Goal**: Add build system, testing, and performance optimizations

---

## 📋 Detailed Implementation Plan

## [ ] PHASE 1: Foundation Improvements

### [ ] TASK-FE-001: Implement Modern Build System
**Priority:** HIGH  
**Effort:** 8 hours  
**Technology:** Vite + TypeScript

**Implementation Plan:**
1. **Setup Vite configuration**
   ```javascript
   // vite.config.js
   import { defineConfig } from 'vite'
   import { resolve } from 'path'
   
   export default defineConfig({
     build: {
       lib: {
         entry: resolve(__dirname, 'web/js/dashboard/index.js'),
         name: 'Dashboard',
         fileName: 'dashboard'
       },
       outDir: 'web/dist',
       sourcemap: true
     },
     server: {
       proxy: {
         '/api': 'http://localhost:3500'
       }
     }
   })
   ```

2. **TypeScript configuration**
   ```json
   // tsconfig.json
   {
     "compilerOptions": {
       "target": "ES2020",
       "module": "ESNext",
       "lib": ["ES2020", "DOM"],
       "strict": true,
       "moduleResolution": "node",
       "allowSyntheticDefaultImports": true
     }
   }
   ```

3. **Package.json scripts**
   ```json
   {
     "scripts": {
       "dev": "vite",
       "build": "vite build",
       "preview": "vite preview",
       "test": "vitest",
       "lint": "eslint web/js --ext .js,.ts"
     }
   }
   ```

**Files to Create:**
- `package.json` - Project dependencies and scripts
- `vite.config.js` - Build configuration
- `tsconfig.json` - TypeScript configuration
- `.eslintrc.js` - Linting rules

**Files to Modify:**
- Update Go templates to reference built assets
- Add development vs production asset loading

### [ ] TASK-FE-002: CSS Architecture Refactoring
**Priority:** HIGH  
**Effort:** 12 hours  
**Technology:** CSS Modules + PostCSS

**Implementation Plan:**
1. **Split monolithic CSS into logical modules**
   ```
   web/styles/
   ├── base/
   │   ├── reset.css
   │   ├── variables.css
   │   └── typography.css
   ├── components/
   │   ├── glass-panel.css
   │   ├── metric-card.css
   │   ├── chart-card.css
   │   ├── heartbeat-item.css
   │   └── server-card.css
   ├── layouts/
   │   ├── dashboard.css
   │   ├── hero.css
   │   └── grid.css
   ├── utilities/
   │   ├── spacing.css
   │   ├── colors.css
   │   └── animations.css
   └── themes/
       ├── dark.css
       └── light.css
   ```

2. **Implement CSS custom properties architecture**
   ```css
   /* variables.css */
   :root {
     /* Semantic color tokens */
     --color-primary: var(--blue-500);
     --color-success: var(--green-500);
     --color-danger: var(--red-500);
     
     /* Component tokens */
     --metric-card-bg: var(--glass-bg);
     --metric-card-border: var(--border);
     --metric-card-radius: var(--radius-md);
   }
   ```

3. **PostCSS configuration**
   ```javascript
   // postcss.config.js
   export default {
     plugins: {
       'postcss-import': {},
       'postcss-nested': {},
       'autoprefixer': {},
       'postcss-custom-properties': {},
       'cssnano': process.env.NODE_ENV === 'production' ? {} : false
     }
   }
   ```

**Files to Create:**
- `web/styles/` directory structure with modular CSS
- `postcss.config.js` - CSS processing configuration
- Component-specific CSS modules

**Files to Modify:**
- Split `web/assets/dashboard.css` into logical modules
- Update import strategy in main CSS file

### [ ] TASK-FE-003: State Management Refactoring
**Priority:** MEDIUM  
**Effort:** 6 hours  
**Technology:** Observer Pattern + Reactive Store

**Implementation Plan:**
1. **Create reactive store implementation**
   ```typescript
   // web/js/store/reactive-store.ts
   interface StoreSubscriber<T> {
     (state: T): void;
   }
   
   export class ReactiveStore<T> {
     private state: T;
     private subscribers: Set<StoreSubscriber<T>> = new Set();
     
     constructor(initialState: T) {
       this.state = initialState;
     }
     
     getState(): T {
       return this.state;
     }
     
     setState(updater: Partial<T> | ((state: T) => T)): void {
       this.state = typeof updater === 'function' 
         ? updater(this.state)
         : { ...this.state, ...updater };
       
       this.subscribers.forEach(subscriber => subscriber(this.state));
     }
     
     subscribe(subscriber: StoreSubscriber<T>): () => void {
       this.subscribers.add(subscriber);
       return () => this.subscribers.delete(subscriber);
     }
   }
   ```

2. **Define typed state interface**
   ```typescript
   // web/js/store/types.ts
   export interface DashboardState {
     // System state
     connectionState: 'online' | 'offline' | 'reconnecting';
     isLoading: boolean;
     lastUpdated: string;
     
     // Data state
     systemData: SystemMonitoring | null;
     serverMetrics: ServerMetrics[];
     heartbeatData: HeartbeatStatus[];
     
     // UI state
     theme: 'dark' | 'light';
     activeFilters: DateFilter;
     selectedServer: string | null;
     
     // Chart state
     chartData: ChartDatasets;
   }
   ```

3. **Create store instance and actions**
   ```typescript
   // web/js/store/dashboard-store.ts
   import { ReactiveStore } from './reactive-store';
   import { DashboardState } from './types';
   
   const initialState: DashboardState = {
     connectionState: 'offline',
     isLoading: true,
     // ... other initial values
   };
   
   export const dashboardStore = new ReactiveStore(initialState);
   
   // Action creators
   export const actions = {
     setConnectionState: (state: 'online' | 'offline' | 'reconnecting') =>
       dashboardStore.setState({ connectionState: state }),
   
     updateSystemData: (data: SystemMonitoring) =>
       dashboardStore.setState({ 
         systemData: data, 
         lastUpdated: new Date().toISOString(),
         isLoading: false 
       }),
   
     toggleTheme: () =>
       dashboardStore.setState(state => ({ 
         theme: state.theme === 'dark' ? 'light' : 'dark' 
       }))
   };
   ```

**Files to Create:**
- `web/js/store/reactive-store.ts` - Reactive store implementation
- `web/js/store/types.ts` - TypeScript type definitions
- `web/js/store/dashboard-store.ts` - Main store instance and actions

**Files to Modify:**
- `web/js/dashboard/state.js` - Migrate to new store system
- All modules using state - Update to use reactive store

---

## [ ] PHASE 2: Alpine.js Integration

### [ ] TASK-FE-004: Alpine.js Foundation Setup
**Priority:** HIGH  
**Effort:** 4 hours  
**Technology:** Alpine.js v3

**Implementation Plan:**
1. **Install and configure Alpine.js**
   ```html
   <!-- Add to layout.templ -->
   <script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js"></script>
   ```

2. **Create Alpine store for global state**
   ```javascript
   // web/js/alpine/stores.js
   import Alpine from 'alpinejs'
   
   Alpine.store('dashboard', {
     // Connection state
     connectionState: 'offline',
     isLoading: true,
     lastUpdated: '--',
     
     // Data
     systemData: null,
     serverMetrics: [],
     
     // UI state
     theme: localStorage.getItem('theme') || 'dark',
     showExportPanel: false,
     
     // Actions
     setConnectionState(state) {
       this.connectionState = state;
     },
     
     updateSystemData(data) {
       this.systemData = data;
       this.lastUpdated = new Date().toLocaleTimeString();
       this.isLoading = false;
     },
     
     toggleTheme() {
       this.theme = this.theme === 'dark' ? 'light' : 'dark';
       localStorage.setItem('theme', this.theme);
       document.documentElement.setAttribute('data-theme', this.theme);
     }
   });
   ```

3. **Initialize Alpine components**
   ```javascript
   // web/js/alpine/init.js
   import Alpine from 'alpinejs'
   import './stores.js'
   import './components/index.js'
   
   // Start Alpine
   Alpine.start()
   ```

**Files to Create:**
- `web/js/alpine/stores.js` - Alpine global stores
- `web/js/alpine/init.js` - Alpine initialization
- `web/js/alpine/components/` - Directory for Alpine components

**Files to Modify:**
- `web/views/layout.templ` - Add Alpine.js script
- `web/js/dashboard/index.js` - Initialize Alpine

### [ ] TASK-FE-005: Convert Metrics Section to Alpine
**Priority:** HIGH  
**Effort:** 8 hours  
**Technology:** Alpine.js + Reactive Templates

**Implementation Plan:**
1. **Update metrics template with Alpine directives**
   ```html
   <!-- web/views/dashboard.templ - MetricsSection -->
   <section class="section-block" 
            x-data="metricsComponent()" 
            data-component="metrics">
     <div class="metrics-grid">
       <!-- CPU Metric Card -->
       <article class="glass-panel metric-card">
         <div class="metric-header">
           <span class="metric-label">CPU Usage</span>
           <span class="metric-trend" 
                 x-text="formatTrend(cpuTrend)" 
                 :class="getTrendClass(cpuTrend)">--</span>
         </div>
         <div class="metric-value-wrap">
           <span class="metric-value" 
                 x-text="formatValue(cpuUsage, 1)" 
                 :class="getValueClass(cpuUsage, 80, 60)">--</span>
           <span class="metric-unit">%</span>
         </div>
         <div class="metric-progress" 
              role="progressbar" 
              :aria-valuenow="cpuUsage">
           <span class="progress-fill" 
                 :style="`width: ${cpuUsage}%`"></span>
         </div>
       </article>
       
       <!-- Memory Metric Card -->
       <article class="glass-panel metric-card">
         <!-- Similar structure for memory -->
       </article>
       
       <!-- Storage Cards - Dynamic -->
       <template x-for="storage in storageDevices" :key="storage.path">
         <article class="glass-panel metric-card storage-card">
           <div class="metric-header">
             <span class="metric-label" x-text="`Storage (${storage.path})`"></span>
           </div>
           <div class="metric-value-wrap">
             <span class="metric-value" 
                   x-text="formatValue(storage.used_pct, 1)">--</span>
             <span class="metric-unit">%</span>
           </div>
           <div class="metric-details">
             <span x-text="formatBytes(storage.used_bytes)"></span> / 
             <span x-text="formatBytes(storage.total_bytes)"></span>
           </div>
         </article>
       </template>
     </div>
   </section>
   ```

2. **Create Alpine metrics component**
   ```javascript
   // web/js/alpine/components/metrics.js
   import Alpine from 'alpinejs'
   
   Alpine.data('metricsComponent', () => ({
     // Reactive properties
     get cpuUsage() {
       return this.$store.dashboard.systemData?.cpu?.usage_percent ?? 0;
     },
     
     get memoryUsage() {
       return this.$store.dashboard.systemData?.ram?.used_pct ?? 0;
     },
     
     get storageDevices() {
       return this.$store.dashboard.systemData?.disk_space ?? [];
     },
     
     get cpuTrend() {
       return this.calculateTrend('cpu_usage');
     },
     
     // Methods
     formatValue(value, decimals = 0) {
       if (value == null || value === undefined) return '--';
       return Number(value).toFixed(decimals);
     },
     
     formatBytes(bytes) {
       if (!bytes) return '--';
       const units = ['B', 'KB', 'MB', 'GB', 'TB'];
       let size = bytes;
       let unitIndex = 0;
       
       while (size >= 1024 && unitIndex < units.length - 1) {
         size /= 1024;
         unitIndex++;
       }
       
       return `${size.toFixed(1)} ${units[unitIndex]}`;
     },
     
     getValueClass(value, critical, warning) {
       if (value >= critical) return 'metric-value--critical';
       if (value >= warning) return 'metric-value--warning';
       return 'metric-value--normal';
     },
     
     getTrendClass(trend) {
       if (trend > 0) return 'metric-trend--up';
       if (trend < 0) return 'metric-trend--down';
       return 'metric-trend--stable';
     },
     
     formatTrend(trend) {
       if (trend > 0) return `↗ +${trend.toFixed(1)}%`;
       if (trend < 0) return `↘ ${trend.toFixed(1)}%`;
       return '→ 0%';
     },
     
     calculateTrend(metric) {
       // Calculate trend based on historical data
       const history = this.$store.dashboard.metricHistory?.[metric] ?? [];
       if (history.length < 2) return 0;
       
       const recent = history.slice(-2);
       return ((recent[1] - recent[0]) / recent[0]) * 100;
     }
   }));
   ```

**Files to Create:**
- `web/js/alpine/components/metrics.js` - Metrics Alpine component
- CSS updates for state-based styling

**Files to Modify:**
- `web/views/dashboard.templ` - Update MetricsSection with Alpine directives
- `web/js/dashboard/metrics.js` - Remove manual DOM manipulation code

### [ ] TASK-FE-006: Convert Connection Status and Theme to Alpine
**Priority:** MEDIUM  
**Effort:** 4 hours  
**Technology:** Alpine.js + Local Storage

**Implementation Plan:**
1. **Update connection status template**
   ```html
   <!-- Chrome Component with Alpine -->
   <div data-component="chrome" 
        x-data="chromeComponent()" 
        style="display: contents;">
     
     <!-- Theme Toggle -->
     <button class="theme-toggle" 
             @click="$store.dashboard.toggleTheme()"
             :aria-label="`Switch to ${$store.dashboard.theme === 'dark' ? 'light' : 'dark'} theme`">
       <i :class="$store.dashboard.theme === 'dark' ? 'fas fa-sun' : 'fas fa-moon'"></i>
     </button>
   
     <!-- Connection Status -->
     <div class="connection-status" 
          :class="{ 'hidden': $store.dashboard.connectionState === 'online' }"
          x-show="$store.dashboard.connectionState !== 'online'"
          x-transition:enter="transition ease-out duration-300"
          x-transition:enter-start="opacity-0 transform scale-95"
          x-transition:enter-end="opacity-100 transform scale-100">
       <span class="status-dot" 
             :class="`status-dot--${$store.dashboard.connectionState}`"></span>
       <span x-text="getConnectionMessage()">Connected</span>
     </div>
   
     <!-- Export Panel -->
     <div class="export-overlay" 
          x-show="$store.dashboard.showExportPanel"
          @click="$store.dashboard.showExportPanel = false"
          x-transition:enter="transition ease-out duration-300"
          x-transition:enter-start="opacity-0">
     </div>
   </div>
   ```

2. **Create Chrome Alpine component**
   ```javascript
   // web/js/alpine/components/chrome.js
   Alpine.data('chromeComponent', () => ({
     init() {
       // Initialize theme on component mount
       this.applyTheme(this.$store.dashboard.theme);
       
       // Watch for theme changes
       this.$watch('$store.dashboard.theme', (newTheme) => {
         this.applyTheme(newTheme);
       });
     },
     
     applyTheme(theme) {
       document.documentElement.setAttribute('data-theme', theme);
       document.body.className = theme === 'dark' ? 'theme-dark' : 'theme-light';
     },
     
     getConnectionMessage() {
       const state = this.$store.dashboard.connectionState;
       const messages = {
         'online': 'Connected',
         'offline': 'Connection lost',
         'reconnecting': 'Reconnecting...'
       };
       return messages[state] || 'Unknown';
     }
   }));
   ```

**Files to Create:**
- `web/js/alpine/components/chrome.js` - Chrome component
- CSS for connection status states

**Files to Modify:**
- `web/views/dashboard.templ` - Update ChromeComponent
- `web/js/dashboard/theme.js` - Remove and replace with Alpine

### [ ] TASK-FE-007: Convert Heartbeat Section to Alpine
**Priority:** MEDIUM  
**Effort:** 6 hours  
**Technology:** Alpine.js + Filtering

**Implementation Plan:**
1. **Update heartbeat template with Alpine**
   ```html
   <!-- Heartbeat Section -->
   <section class="glass-panel heartbeat-section" 
            x-data="heartbeatComponent()" 
            data-component="heartbeats">
     <div class="heartbeat-header">
       <div>
         <div class="heartbeat-title">Domain Heartbeat</div>
         <div class="heartbeat-meta">Live status across configured targets</div>
       </div>
       <div class="heartbeat-meta" 
            x-text="`${onlineCount} online / ${totalCount} total`">
         -- online / -- total
       </div>
     </div>
     
     <div class="heartbeat-controls">
       <input type="text" 
              class="heartbeat-search"
              placeholder="Search servers..."
              x-model="searchQuery"
              @input="filterHeartbeats">
     </div>
     
     <div class="heartbeat-grid">
       <template x-for="server in filteredServers" :key="server.name">
         <div class="heartbeat-item" 
              :class="`heartbeat-item--${server.status}`">
           <div class="heartbeat-status">
             <span class="status-indicator" 
                   :class="`status-indicator--${server.status}`">
               <i :class="server.status === 'up' ? 'fas fa-check' : 'fas fa-times'"></i>
             </span>
             <span class="status-text" 
                   x-text="server.status.toUpperCase()">UP</span>
           </div>
           
           <div class="heartbeat-details">
             <div class="heartbeat-name" x-text="server.name">Server Name</div>
             <div class="heartbeat-url" x-text="server.url">https://example.com</div>
           </div>
           
           <div class="heartbeat-metrics">
             <span class="response-time" 
                   x-text="server.response_time">--ms</span>
             <span class="last-checked" 
                   x-text="formatLastChecked(server.last_checked)">Just now</span>
           </div>
         </div>
       </template>
       
       <div x-show="filteredServers.length === 0" 
            class="heartbeat-empty">
         <span x-text="searchQuery ? 'No matching servers found' : 'No heartbeat data yet'">
           No heartbeat data yet
         </span>
       </div>
     </div>
   </section>
   ```

2. **Create heartbeat Alpine component**
   ```javascript
   // web/js/alpine/components/heartbeat.js
   Alpine.data('heartbeatComponent', () => ({
     searchQuery: '',
     
     get heartbeatData() {
       return this.$store.dashboard.systemData?.heartbeat ?? [];
     },
     
     get filteredServers() {
       if (!this.searchQuery) return this.heartbeatData;
       
       const query = this.searchQuery.toLowerCase();
       return this.heartbeatData.filter(server => 
         server.name.toLowerCase().includes(query) ||
         server.url.toLowerCase().includes(query)
       );
     },
     
     get onlineCount() {
       return this.heartbeatData.filter(s => s.status === 'up').length;
     },
     
     get totalCount() {
       return this.heartbeatData.length;
     },
     
     formatLastChecked(timestamp) {
       if (!timestamp) return '--';
       const now = new Date();
       const checked = new Date(timestamp);
       const diffMs = now - checked;
       
       if (diffMs < 60000) return 'Just now';
       if (diffMs < 3600000) return `${Math.floor(diffMs / 60000)}m ago`;
       return `${Math.floor(diffMs / 3600000)}h ago`;
     }
   }));
   ```

**Files to Create:**
- `web/js/alpine/components/heartbeat.js` - Heartbeat component
- CSS for heartbeat status states

**Files to Modify:**
- `web/views/dashboard.templ` - Update HeartbeatSection
- `web/js/dashboard/heartbeat.js` - Remove manual DOM logic

---

## [ ] PHASE 3: Web Components with Lit

### [ ] TASK-FE-008: Chart Components with Lit
**Priority:** HIGH  
**Effort:** 12 hours  
**Technology:** Lit + Chart.js

**Implementation Plan:**
1. **Create base chart component**
   ```typescript
   // web/js/components/base-chart.ts
   import { LitElement, html, css, PropertyValues } from 'lit';
   import { customElement, property, state } from 'lit/decorators.js';
   import { Chart, ChartConfiguration } from 'chart.js/auto';
   
   @customElement('base-chart')
   export class BaseChart extends LitElement {
     @property({ type: Object }) chartConfig!: ChartConfiguration;
     @property({ type: Object }) data = {};
     @state() private chart?: Chart;
     
     static styles = css`
       :host {
         display: block;
         position: relative;
         height: 100%;
         width: 100%;
       }
       
       canvas {
         max-height: 100%;
         max-width: 100%;
       }
     `;
     
     render() {
       return html`<canvas></canvas>`;
     }
     
     firstUpdated() {
       this.initChart();
     }
     
     updated(changedProperties: PropertyValues) {
       if (changedProperties.has('data') && this.chart) {
         this.updateChart();
       }
     }
     
     private initChart() {
       const canvas = this.shadowRoot?.querySelector('canvas');
       if (!canvas) return;
       
       this.chart = new Chart(canvas, this.chartConfig);
     }
     
     private updateChart() {
       if (!this.chart) return;
       
       // Update chart data based on this.data
       this.chart.data = this.transformData(this.data);
       this.chart.update('none');
     }
     
     private transformData(data: any) {
       // Override in specific chart components
       return data;
     }
   }
   ```

2. **Create system performance chart component**
   ```typescript
   // web/js/components/system-chart.ts
   import { customElement, property } from 'lit/decorators.js';
   import { BaseChart } from './base-chart.js';
   import { ChartConfiguration } from 'chart.js/auto';
   
   @customElement('system-chart')
   export class SystemChart extends BaseChart {
     @property({ type: Array }) systemHistory = [];
     
     chartConfig: ChartConfiguration = {
       type: 'line',
       data: {
         labels: [],
         datasets: [
           {
             label: 'CPU Usage (%)',
             data: [],
             borderColor: 'rgba(56, 189, 248, 1)',
             backgroundColor: 'rgba(56, 189, 248, 0.1)',
             tension: 0.4
           },
           {
             label: 'Memory Usage (%)',
             data: [],
             borderColor: 'rgba(34, 197, 94, 1)',
             backgroundColor: 'rgba(34, 197, 94, 0.1)',
             tension: 0.4
           }
         ]
       },
       options: {
         responsive: true,
         maintainAspectRatio: false,
         plugins: {
           legend: {
             position: 'top',
             labels: {
               font: { family: 'Inter', size: 12 },
               color: 'var(--text-secondary)'
             }
           }
         },
         scales: {
           x: {
             grid: { color: 'var(--border)' },
             ticks: { color: 'var(--text-secondary)' }
           },
           y: {
             beginAtZero: true,
             max: 100,
             grid: { color: 'var(--border)' },
             ticks: { color: 'var(--text-secondary)' }
           }
         }
       }
     };
     
     protected transformData(data: any) {
       const now = new Date();
       const labels = Array.from({ length: 20 }, (_, i) => {
         const time = new Date(now.getTime() - (19 - i) * 5000);
         return time.toLocaleTimeString([], { 
           hour12: false, 
           minute: '2-digit', 
           second: '2-digit' 
         });
       });
       
       return {
         labels,
         datasets: [
           {
             ...this.chartConfig.data!.datasets![0],
             data: this.systemHistory.map(h => h.cpu_usage || 0)
           },
           {
             ...this.chartConfig.data!.datasets![1],
             data: this.systemHistory.map(h => h.memory_usage || 0)
           }
         ]
       };
     }
   }
   ```

3. **Create network chart component**
   ```typescript
   // web/js/components/network-chart.ts
   import { customElement } from 'lit/decorators.js';
   import { BaseChart } from './base-chart.js';
   
   @customElement('network-chart')
   export class NetworkChart extends BaseChart {
     chartConfig = {
       type: 'line' as const,
       data: {
         labels: [],
         datasets: [
           {
             label: 'Inbound (MB/s)',
             data: [],
             borderColor: 'rgba(129, 140, 248, 1)',
             backgroundColor: 'rgba(129, 140, 248, 0.1)',
             fill: true
           },
           {
             label: 'Outbound (MB/s)',
             data: [],
             borderColor: 'rgba(245, 101, 101, 1)',
             backgroundColor: 'rgba(245, 101, 101, 0.1)',
             fill: true
           }
         ]
       },
       options: {
         responsive: true,
         maintainAspectRatio: false,
         interaction: { intersect: false },
         scales: {
           x: { 
             grid: { color: 'var(--border)' },
             ticks: { color: 'var(--text-secondary)' }
           },
           y: {
             beginAtZero: true,
             grid: { color: 'var(--border)' },
             ticks: { 
               color: 'var(--text-secondary)',
               callback: (value: any) => `${value} MB/s`
             }
           }
         }
       }
     };
   }
   ```

4. **Update templates to use Web Components**
   ```html
   <!-- Charts Section -->
   <section class="section-block" data-component="charts">
     <div class="charts-grid">
       <article class="glass-panel chart-card">
         <div class="chart-header">
           <h3>System Performance</h3>
           <span class="chart-subtitle">CPU and memory usage over time</span>
         </div>
         <div class="chart-wrapper">
           <system-chart 
             :system-history="$store.dashboard.systemHistory">
           </system-chart>
         </div>
       </article>
       
       <article class="glass-panel chart-card">
         <div class="chart-header">
           <h3>Network Throughput</h3>
           <span class="chart-subtitle">Inbound vs outbound throughput</span>
         </div>
         <div class="chart-wrapper">
           <network-chart 
             :network-history="$store.dashboard.networkHistory">
           </network-chart>
         </div>
       </article>
     </div>
   </section>
   ```

**Files to Create:**
- `web/js/components/base-chart.ts` - Base chart component
- `web/js/components/system-chart.ts` - System performance chart
- `web/js/components/network-chart.ts` - Network throughput chart
- `web/js/components/usage-donut.ts` - Resource usage donut chart

**Files to Modify:**
- `web/views/dashboard.templ` - Update chart sections
- `web/js/dashboard/charts.js` - Remove manual chart logic

### [ ] TASK-FE-009: Server Metrics Component
**Priority:** MEDIUM  
**Effort:** 8 hours  
**Technology:** Lit + Virtual Scrolling

**Implementation Plan:**
1. **Create server card component**
   ```typescript
   // web/js/components/server-card.ts
   import { LitElement, html, css } from 'lit';
   import { customElement, property } from 'lit/decorators.js';
   
   interface ServerMetrics {
     name: string;
     address: string;
     cpu_usage: number;
     memory_used_percent: number;
     disk_used_percent: number;
     status: 'ok' | 'error' | 'warning';
     timestamp: string;
   }
   
   @customElement('server-card')
   export class ServerCard extends LitElement {
     @property({ type: Object }) server!: ServerMetrics;
     @property({ type: Boolean }) selected = false;
     
     static styles = css`
       :host {
         display: block;
         cursor: pointer;
         transition: all var(--transition-base);
       }
       
       :host(:hover) {
         transform: translateY(-2px);
       }
       
       :host([selected]) .server-card {
         border-color: var(--accent);
         box-shadow: 0 0 0 1px var(--accent);
       }
       
       .server-card {
         background: var(--glass-bg);
         border: 1px solid var(--border);
         border-radius: var(--radius-md);
         padding: 1.5rem;
         backdrop-filter: blur(var(--glass-blur));
       }
       
       .server-header {
         display: flex;
         justify-content: space-between;
         align-items: center;
         margin-bottom: 1rem;
       }
       
       .server-name {
         font-weight: 600;
         color: var(--text-primary);
       }
       
       .server-status {
         padding: 0.25rem 0.75rem;
         border-radius: var(--radius-pill);
         font-size: 0.75rem;
         font-weight: 500;
         text-transform: uppercase;
       }
       
       .server-status--ok {
         background: var(--success);
         color: white;
       }
       
       .server-status--error {
         background: var(--danger);
         color: white;
       }
       
       .server-status--warning {
         background: var(--warning);
         color: white;
       }
       
       .metrics-grid {
         display: grid;
         grid-template-columns: repeat(3, 1fr);
         gap: 1rem;
         margin-bottom: 1rem;
       }
       
       .metric-item {
         text-align: center;
       }
       
       .metric-label {
         font-size: 0.75rem;
         color: var(--text-secondary);
         margin-bottom: 0.25rem;
       }
       
       .metric-value {
         font-size: 1.25rem;
         font-weight: 600;
         color: var(--text-primary);
       }
       
       .server-footer {
         font-size: 0.75rem;
         color: var(--text-secondary);
         display: flex;
         justify-content: space-between;
         align-items: center;
       }
     `;
     
     render() {
       return html`
         <div class="server-card" @click=${this.handleClick}>
           <div class="server-header">
             <div class="server-name">${this.server.name}</div>
             <div class="server-status server-status--${this.server.status}">
               ${this.server.status}
             </div>
           </div>
           
           <div class="metrics-grid">
             <div class="metric-item">
               <div class="metric-label">CPU</div>
               <div class="metric-value">
                 ${this.formatPercent(this.server.cpu_usage)}%
               </div>
             </div>
             <div class="metric-item">
               <div class="metric-label">Memory</div>
               <div class="metric-value">
                 ${this.formatPercent(this.server.memory_used_percent)}%
               </div>
             </div>
             <div class="metric-item">
               <div class="metric-label">Disk</div>
               <div class="metric-value">
                 ${this.formatPercent(this.server.disk_used_percent)}%
               </div>
             </div>
           </div>
           
           <div class="server-footer">
             <span class="server-address">${this.server.address}</span>
             <span class="server-timestamp">
               ${this.formatTimestamp(this.server.timestamp)}
             </span>
           </div>
         </div>
       `;
     }
     
     private handleClick() {
       this.dispatchEvent(new CustomEvent('server-select', {
         detail: { server: this.server },
         bubbles: true
       }));
     }
     
     private formatPercent(value: number): string {
       return value?.toFixed(1) ?? '--';
     }
     
     private formatTimestamp(timestamp: string): string {
       if (!timestamp) return '--';
       return new Date(timestamp).toLocaleTimeString();
     }
   }
   ```

2. **Create server list component**
   ```typescript
   // web/js/components/server-list.ts
   import { LitElement, html, css } from 'lit';
   import { customElement, property, state } from 'lit/decorators.js';
   import './server-card.js';
   
   @customElement('server-list')
   export class ServerList extends LitElement {
     @property({ type: Array }) servers = [];
     @property({ type: String }) selectedServer = '';
     @state() private searchQuery = '';
     
     static styles = css`
       .servers-header {
         display: flex;
         justify-content: space-between;
         align-items: center;
         margin-bottom: 1.5rem;
       }
       
       .servers-title {
         font-size: 1.25rem;
         font-weight: 600;
         color: var(--text-primary);
       }
       
       .servers-count {
         color: var(--text-secondary);
       }
       
       .servers-controls {
         margin-bottom: 1.5rem;
       }
       
       .search-input {
         width: 100%;
         padding: 0.75rem 1rem;
         border: 1px solid var(--border);
         border-radius: var(--radius-sm);
         background: var(--glass-bg);
         color: var(--text-primary);
         font-size: 0.875rem;
       }
       
       .servers-grid {
         display: grid;
         grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
         gap: 1.5rem;
       }
       
       .servers-empty {
         text-align: center;
         color: var(--text-secondary);
         padding: 3rem;
         font-style: italic;
       }
     `;
     
     render() {
       const filteredServers = this.getFilteredServers();
       
       return html`
         <div class="servers-header">
           <div>
             <div class="servers-title">Remote Servers</div>
             <div class="servers-meta">Latest health signals from configured servers</div>
           </div>
           <div class="servers-count">
             ${filteredServers.length} server${filteredServers.length !== 1 ? 's' : ''} tracked
           </div>
         </div>
         
         <div class="servers-controls">
           <input 
             type="text" 
             class="search-input"
             placeholder="Search servers by name or address..."
             @input=${this.handleSearch}
             .value=${this.searchQuery}>
         </div>
         
         <div class="servers-grid">
           ${filteredServers.length > 0
             ? filteredServers.map(server => html`
                 <server-card 
                   .server=${server}
                   ?selected=${server.address === this.selectedServer}
                   @server-select=${this.handleServerSelect}>
                 </server-card>
               `)
             : html`<div class="servers-empty">No servers found</div>`
           }
         </div>
       `;
     }
     
     private getFilteredServers() {
       if (!this.searchQuery) return this.servers;
       
       const query = this.searchQuery.toLowerCase();
       return this.servers.filter(server => 
         server.name.toLowerCase().includes(query) ||
         server.address.toLowerCase().includes(query)
       );
     }
     
     private handleSearch(e: Event) {
       this.searchQuery = (e.target as HTMLInputElement).value;
     }
     
     private handleServerSelect(e: CustomEvent) {
       this.dispatchEvent(new CustomEvent('server-change', {
         detail: e.detail,
         bubbles: true
       }));
     }
   }
   ```

**Files to Create:**
- `web/js/components/server-card.ts` - Individual server card
- `web/js/components/server-list.ts` - Server list container
- CSS for server components

**Files to Modify:**
- `web/views/dashboard.templ` - Update servers section
- `web/js/dashboard/servers.js` - Remove manual server logic

---

## [ ] PHASE 4: Advanced Features & Optimization

### [ ] TASK-FE-010: Implement Advanced State Management
**Priority:** MEDIUM  
**Effort:** 10 hours  
**Technology:** Zustand-like Store + TypeScript

**Implementation Plan:**
1. **Create advanced reactive store with middleware**
   ```typescript
   // web/js/store/advanced-store.ts
   interface StoreMiddleware<T> {
     (state: T, action: string, payload: any): T;
   }
   
   interface StoreSubscription<T> {
     selector: (state: T) => any;
     callback: (selected: any, state: T) => void;
   }
   
   export class AdvancedStore<T extends Record<string, any>> {
     private state: T;
     private subscriptions: Set<StoreSubscription<T>> = new Set();
     private middleware: StoreMiddleware<T>[] = [];
     private history: T[] = [];
     
     constructor(initialState: T) {
       this.state = initialState;
       this.history.push(initialState);
     }
     
     // Middleware system
     use(middleware: StoreMiddleware<T>) {
       this.middleware.push(middleware);
       return this;
     }
     
     // Computed selectors
     select<K>(selector: (state: T) => K): K {
       return selector(this.state);
     }
     
     // Subscribe to state changes with selectors
     subscribe<K>(
       selector: (state: T) => K,
       callback: (selected: K, state: T) => void,
       options: { immediate?: boolean } = {}
     ) {
       const subscription = { selector, callback };
       this.subscriptions.add(subscription);
       
       if (options.immediate) {
         callback(selector(this.state), this.state);
       }
       
       return () => this.subscriptions.delete(subscription);
     }
     
     // Action dispatch with middleware
     dispatch(action: string, payload?: any) {
       let newState = this.state;
       
       // Apply middleware
       for (const middleware of this.middleware) {
         newState = middleware(newState, action, payload);
       }
       
       if (newState !== this.state) {
         this.state = newState;
         this.history.push(newState);
         
         // Limit history size
         if (this.history.length > 50) {
           this.history.shift();
         }
         
         this.notifySubscribers();
       }
     }
     
     private notifySubscribers() {
       this.subscriptions.forEach(({ selector, callback }) => {
         try {
           const selected = selector(this.state);
           callback(selected, this.state);
         } catch (error) {
           console.error('Subscription callback error:', error);
         }
       });
     }
     
     // Time travel debugging
     revertToState(index: number) {
       if (index >= 0 && index < this.history.length) {
         this.state = this.history[index];
         this.notifySubscribers();
       }
     }
     
     getHistory() {
       return [...this.history];
     }
   }
   ```

2. **Create typed actions and reducers**
   ```typescript
   // web/js/store/dashboard-actions.ts
   export type DashboardAction = 
     | { type: 'SET_CONNECTION_STATE'; payload: 'online' | 'offline' | 'reconnecting' }
     | { type: 'UPDATE_SYSTEM_DATA'; payload: SystemMonitoring }
     | { type: 'SET_THEME'; payload: 'dark' | 'light' }
     | { type: 'SET_LOADING'; payload: boolean }
     | { type: 'ADD_ALERT'; payload: Alert }
     | { type: 'REMOVE_ALERT'; payload: string }
     | { type: 'SELECT_SERVER'; payload: string | null };
   
   export const dashboardReducer = (
     state: DashboardState, 
     action: string, 
     payload: any
   ): DashboardState => {
     switch (action) {
       case 'SET_CONNECTION_STATE':
         return { ...state, connectionState: payload };
         
       case 'UPDATE_SYSTEM_DATA':
         return {
           ...state,
           systemData: payload,
           lastUpdated: new Date().toISOString(),
           isLoading: false,
           // Add to history for trending
           systemHistory: [
             ...state.systemHistory.slice(-19),
             {
               timestamp: payload.timestamp,
               cpu_usage: payload.cpu.usage_percent,
               memory_usage: payload.ram.used_pct,
               network_rx: payload.network_io.bytes_recv,
               network_tx: payload.network_io.bytes_sent
             }
           ]
         };
         
       case 'SET_THEME':
         localStorage.setItem('theme', payload);
         document.documentElement.setAttribute('data-theme', payload);
         return { ...state, theme: payload };
         
       case 'ADD_ALERT':
         return {
           ...state,
           alerts: [...state.alerts, { ...payload, id: Date.now().toString() }]
         };
         
       case 'REMOVE_ALERT':
         return {
           ...state,
           alerts: state.alerts.filter(alert => alert.id !== payload)
         };
         
       default:
         return state;
     }
   };
   ```

3. **Add development middleware**
   ```typescript
   // web/js/store/middleware.ts
   export const loggingMiddleware: StoreMiddleware<DashboardState> = (
     state, 
     action, 
     payload
   ) => {
     if (process.env.NODE_ENV === 'development') {
       console.group(`Action: ${action}`);
       console.log('Previous State:', state);
       console.log('Payload:', payload);
       
       const newState = dashboardReducer(state, action, payload);
       console.log('Next State:', newState);
       console.groupEnd();
       
       return newState;
     }
     
     return dashboardReducer(state, action, payload);
   };
   
   export const persistenceMiddleware: StoreMiddleware<DashboardState> = (
     state,
     action,
     payload
   ) => {
     const newState = dashboardReducer(state, action, payload);
     
     // Persist certain state to localStorage
     if (['SET_THEME', 'SET_FILTERS'].includes(action)) {
       const persistedState = {
         theme: newState.theme,
         filters: newState.activeFilters
       };
       localStorage.setItem('dashboard-state', JSON.stringify(persistedState));
     }
     
     return newState;
   };
   ```

**Files to Create:**
- `web/js/store/advanced-store.ts` - Advanced store implementation
- `web/js/store/dashboard-actions.ts` - Typed actions and reducers
- `web/js/store/middleware.ts` - Store middleware
- `web/js/store/selectors.ts` - Memoized selectors

**Files to Modify:**
- All components to use new store system
- Alpine stores to integrate with advanced store

### [ ] TASK-FE-011: Add Testing Infrastructure
**Priority:** HIGH  
**Effort:** 12 hours  
**Technology:** Vitest + Testing Library + Playwright

**Implementation Plan:**
1. **Setup testing configuration**
   ```javascript
   // vitest.config.js
   import { defineConfig } from 'vitest/config'
   
   export default defineConfig({
     test: {
       environment: 'jsdom',
       setupFiles: ['./test/setup.ts'],
       coverage: {
         provider: 'c8',
         reporter: ['text', 'html', 'lcov'],
         exclude: [
           'node_modules/',
           'test/',
           '**/*.d.ts',
           'web/js/components/**/*.stories.ts'
         ]
       }
     },
     resolve: {
       alias: {
         '@': '/web/js'
       }
     }
   })
   ```

2. **Create test utilities**
   ```typescript
   // test/utils.ts
   import { render, RenderOptions } from '@testing-library/lit';
   import { LitElement } from 'lit';
   
   // Mock store for testing
   export const createMockStore = (initialState = {}) => ({
     getState: () => initialState,
     setState: vi.fn(),
     subscribe: vi.fn(() => () => {}),
     dispatch: vi.fn()
   });
   
   // Custom render function with store context
   export const renderWithStore = (
     component: LitElement, 
     options: RenderOptions & { store?: any } = {}
   ) => {
     const { store = createMockStore(), ...renderOptions } = options;
     
     // Mock global store
     (window as any).dashboardStore = store;
     
     return render(component, renderOptions);
   };
   
   // Mock data generators
   export const mockSystemData = {
     timestamp: new Date().toISOString(),
     cpu: { usage_percent: 45.2, core_count: 8 },
     ram: { used_pct: 67.8, total_bytes: 16000000000 },
     disk_space: [
       { path: '/', used_pct: 78.3, total_bytes: 500000000000 }
     ],
     network_io: { bytes_sent: 1234567, bytes_recv: 9876543 },
     heartbeat: [
       { name: 'Test Server', url: 'https://test.com', status: 'up' }
     ]
   };
   ```

3. **Component tests**
   ```typescript
   // test/components/server-card.test.ts
   import { expect, test, vi } from 'vitest';
   import { screen } from '@testing-library/lit';
   import { userEvent } from '@testing-library/user-event';
   import { ServerCard } from '@/components/server-card.js';
   import { renderWithStore } from '../utils.js';
   
   test('renders server information correctly', async () => {
     const mockServer = {
       name: 'Test Server',
       address: 'https://test.example.com',
       cpu_usage: 45.2,
       memory_used_percent: 67.8,
       disk_used_percent: 23.1,
       status: 'ok' as const,
       timestamp: new Date().toISOString()
     };
     
     const serverCard = new ServerCard();
     serverCard.server = mockServer;
     
     renderWithStore(serverCard);
     
     expect(screen.getByText('Test Server')).toBeInTheDocument();
     expect(screen.getByText('45.2%')).toBeInTheDocument();
     expect(screen.getByText('67.8%')).toBeInTheDocument();
     expect(screen.getByText('OK')).toBeInTheDocument();
   });
   
   test('emits server-select event when clicked', async () => {
     const mockServer = { name: 'Test', address: 'test.com', status: 'ok' };
     const selectHandler = vi.fn();
     
     const serverCard = new ServerCard();
     serverCard.server = mockServer;
     serverCard.addEventListener('server-select', selectHandler);
     
     renderWithStore(serverCard);
     
     await userEvent.click(screen.getByRole('button'));
     
     expect(selectHandler).toHaveBeenCalledWith(
       expect.objectContaining({
         detail: { server: mockServer }
       })
     );
   });
   ```

4. **Integration tests**
   ```typescript
   // test/integration/dashboard.test.ts
   import { expect, test, beforeEach } from 'vitest';
   import { page } from '@playwright/test';
   
   test.describe('Dashboard Integration', () => {
     beforeEach(async () => {
       await page.goto('http://localhost:3500');
     });
     
     test('loads dashboard and displays metrics', async () => {
       await expect(page.locator('[data-component="metrics"]')).toBeVisible();
       await expect(page.locator('.metric-value')).toContainText(/\d+/);
     });
     
     test('theme toggle works correctly', async () => {
       const themeToggle = page.locator('#themeToggle');
       await themeToggle.click();
       
       await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
       
       await themeToggle.click();
       await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
     });
     
     test('date filtering updates data', async () => {
       const fromInput = page.locator('#filterFrom');
       const toInput = page.locator('#filterTo');
       const applyButton = page.locator('#applyFilter');
       
       await fromInput.fill('2024-01-01T00:00');
       await toInput.fill('2024-01-31T23:59');
       await applyButton.click();
       
       // Wait for API call to complete
       await page.waitForResponse('**/monitoring');
       
       await expect(page.locator('#lastUpdated')).toContainText(/\d/);
     });
   });
   ```

**Files to Create:**
- `vitest.config.js` - Test configuration
- `test/setup.ts` - Test setup and mocks
- `test/utils.ts` - Test utilities and helpers
- `test/components/*.test.ts` - Component unit tests
- `test/integration/*.test.ts` - Integration tests
- `playwright.config.ts` - E2E test configuration

**Files to Modify:**
- `package.json` - Add test scripts and dependencies
- Add test commands to CI/CD pipeline

### [ ] TASK-FE-012: Performance Optimization
**Priority:** MEDIUM  
**Effort:** 8 hours  
**Technology:** Virtual Scrolling + Lazy Loading + Web Workers

**Implementation Plan:**
1. **Implement virtual scrolling for large lists**
   ```typescript
   // web/js/components/virtual-list.ts
   import { LitElement, html, css } from 'lit';
   import { customElement, property, state } from 'lit/decorators.js';
   
   @customElement('virtual-list')
   export class VirtualList extends LitElement {
     @property({ type: Array }) items = [];
     @property({ type: Number }) itemHeight = 80;
     @property({ type: Number }) containerHeight = 400;
     @property({ type: Function }) renderItem!: (item: any, index: number) => any;
     
     @state() private scrollTop = 0;
     @state() private visibleRange = { start: 0, end: 0 };
     
     static styles = css`
       .virtual-container {
         height: var(--container-height);
         overflow-y: auto;
         position: relative;
       }
       
       .virtual-content {
         position: relative;
       }
       
       .virtual-item {
         position: absolute;
         left: 0;
         right: 0;
       }
     `;
     
     render() {
       const totalHeight = this.items.length * this.itemHeight;
       const visibleItems = this.getVisibleItems();
       
       return html`
         <div 
           class="virtual-container"
           style="--container-height: ${this.containerHeight}px"
           @scroll=${this.handleScroll}>
           <div 
             class="virtual-content" 
             style="height: ${totalHeight}px">
             ${visibleItems.map(({ item, index }) => html`
               <div 
                 class="virtual-item"
                 style="top: ${index * this.itemHeight}px; height: ${this.itemHeight}px">
                 ${this.renderItem(item, index)}
               </div>
             `)}
           </div>
         </div>
       `;
     }
     
     private handleScroll(e: Event) {
       this.scrollTop = (e.target as HTMLElement).scrollTop;
       this.updateVisibleRange();
     }
     
     private updateVisibleRange() {
       const start = Math.floor(this.scrollTop / this.itemHeight);
       const visibleCount = Math.ceil(this.containerHeight / this.itemHeight);
       const buffer = 3; // Render extra items for smooth scrolling
       
       this.visibleRange = {
         start: Math.max(0, start - buffer),
         end: Math.min(this.items.length, start + visibleCount + buffer)
       };
     }
     
     private getVisibleItems() {
       return this.items
         .slice(this.visibleRange.start, this.visibleRange.end)
         .map((item, i) => ({
           item,
           index: this.visibleRange.start + i
         }));
     }
   }
   ```

2. **Add Web Worker for data processing**
   ```typescript
   // web/js/workers/data-processor.ts
   export interface DataProcessorMessage {
     type: 'PROCESS_METRICS' | 'CALCULATE_TRENDS' | 'AGGREGATE_DATA';
     payload: any;
   }
   
   self.onmessage = (event: MessageEvent<DataProcessorMessage>) => {
     const { type, payload } = event.data;
     
     switch (type) {
       case 'PROCESS_METRICS':
         processMetrics(payload);
         break;
         
       case 'CALCULATE_TRENDS':
         calculateTrends(payload);
         break;
         
       case 'AGGREGATE_DATA':
         aggregateData(payload);
         break;
     }
   };
   
   function processMetrics(data: any[]) {
     const processed = data.map(item => ({
       ...item,
       cpu_trend: calculateTrend(item.cpu_history),
       memory_trend: calculateTrend(item.memory_history),
       network_rate: calculateNetworkRate(item.network_history)
     }));
     
     self.postMessage({
       type: 'METRICS_PROCESSED',
       payload: processed
     });
   }
   
   function calculateTrend(history: number[]): number {
     if (history.length < 2) return 0;
     
     const recent = history.slice(-10);
     const first = recent[0];
     const last = recent[recent.length - 1];
     
     return ((last - first) / first) * 100;
   }
   
   function calculateNetworkRate(history: any[]): { rx: number; tx: number } {
     if (history.length < 2) return { rx: 0, tx: 0 };
     
     const prev = history[history.length - 2];
     const curr = history[history.length - 1];
     const timeDiff = (new Date(curr.timestamp).getTime() - 
                      new Date(prev.timestamp).getTime()) / 1000;
     
     return {
       rx: (curr.bytes_recv - prev.bytes_recv) / timeDiff,
       tx: (curr.bytes_sent - prev.bytes_sent) / timeDiff
     };
   }
   ```

3. **Implement lazy loading for components**
   ```typescript
   // web/js/utils/lazy-loader.ts
   export class LazyLoader {
     private static observers = new Map<Element, IntersectionObserver>();
     
     static observe(
       element: Element, 
       callback: () => void, 
       options: IntersectionObserverInit = {}
     ) {
       const observer = new IntersectionObserver((entries) => {
         entries.forEach(entry => {
           if (entry.isIntersecting) {
             callback();
             observer.unobserve(element);
             this.observers.delete(element);
           }
         });
       }, {
         threshold: 0.1,
         rootMargin: '50px',
         ...options
       });
       
       observer.observe(element);
       this.observers.set(element, observer);
     }
     
     static unobserve(element: Element) {
       const observer = this.observers.get(element);
       if (observer) {
         observer.unobserve(element);
         this.observers.delete(element);
       }
     }
   }
   
   // Lazy chart component
   @customElement('lazy-chart')
   export class LazyChart extends LitElement {
     @property({ type: Object }) chartConfig = {};
     @state() private loaded = false;
     
     connectedCallback() {
       super.connectedCallback();
       
       LazyLoader.observe(this, () => {
         this.loaded = true;
       });
     }
     
     disconnectedCallback() {
       super.disconnectedCallback();
       LazyLoader.unobserve(this);
     }
     
     render() {
       if (!this.loaded) {
         return html`
           <div class="chart-placeholder">
             <div class="loading-spinner"></div>
             <span>Loading chart...</span>
           </div>
         `;
       }
       
       return html`<base-chart .chartConfig=${this.chartConfig}></base-chart>`;
     }
   }
   ```

**Files to Create:**
- `web/js/components/virtual-list.ts` - Virtual scrolling component
- `web/js/workers/data-processor.ts` - Web Worker for data processing
- `web/js/utils/lazy-loader.ts` - Lazy loading utilities
- `web/js/components/lazy-chart.ts` - Lazy-loaded chart component

**Files to Modify:**
- Large list components to use virtual scrolling
- Chart components to implement lazy loading
- Data processing to use Web Workers

---

## 🎯 Implementation Roadmap

### Week 1-2: Foundation (PHASE 1)
- [ ] **Day 1-2**: Setup build system (Vite + TypeScript)
- [ ] **Day 3-5**: Refactor CSS architecture (CSS modules)
- [ ] **Day 6-7**: Implement reactive store

### Week 3-4: Reactive Framework (PHASE 2)
- [ ] **Day 8-9**: Alpine.js setup and metrics conversion
- [ ] **Day 10-11**: Connection status and theme Alpine components
- [ ] **Day 12-14**: Heartbeat section Alpine conversion

### Week 5-6: Web Components (PHASE 3)
- [ ] **Day 15-17**: Chart components with Lit
- [ ] **Day 18-21**: Server metrics components

### Week 7-8: Advanced Features (PHASE 4)
- [ ] **Day 22-24**: Advanced state management
- [ ] **Day 25-27**: Testing infrastructure
- [ ] **Day 28**: Performance optimizations

---

## 📊 Expected Benefits

### Code Quality Improvements
- **60% reduction** in manual DOM manipulation
- **40% reduction** in total JavaScript code
- **80% reduction** in state-related bugs
- **90% improvement** in maintainability score

### Performance Improvements
- **30% faster** initial page load (with build optimization)
- **50% smoother** UI updates (with virtual DOM)
- **70% better** mobile performance
- **40% smaller** bundle size (with tree shaking)

### Developer Experience
- **TypeScript** for type safety and better IDE support
- **Hot Module Replacement** for faster development
- **Component testing** for reliable code
- **Modern debugging** tools and DevTools integration

### User Experience
- **Smoother animations** and transitions
- **Better responsiveness** on mobile devices
- **Improved accessibility** with proper ARIA attributes
- **Faster interactions** with optimized rendering

---

## 🔧 Migration Strategy

### Gradual Migration Approach
1. **Start with isolated components** (metrics cards, theme toggle)
2. **Maintain backward compatibility** throughout migration
3. **Test thoroughly** at each phase
4. **Rollback plan** for each component

### Risk Mitigation
- **Feature flags** for new components
- **A/B testing** for critical sections
- **Performance monitoring** during rollout
- **User feedback collection** for UX validation

### Success Metrics
- **Bundle size**: Target <200KB total
- **Performance**: Lighthouse score >90
- **Test coverage**: >80% for all components
- **Developer satisfaction**: Measured via team feedback

This modernization plan transforms the existing vanilla JavaScript implementation into a modern, reactive, and maintainable frontend architecture while preserving the excellent design system and functionality already in place.