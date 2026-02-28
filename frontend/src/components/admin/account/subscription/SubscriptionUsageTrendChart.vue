<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { Chart, registerables } from 'chart.js'

Chart.register(...registerables)

interface AccountTrend {
  accountId: number
  accountName: string
  data: { date: string; usage: number }[]
}

interface Props {
  trends: AccountTrend[]
}

const props = defineProps<Props>()

const chartCanvas = ref<HTMLCanvasElement | null>(null)
let chartInstance: Chart | null = null

// 颜色列表
const colors = [
  'rgb(59, 130, 246)',   // blue
  'rgb(16, 185, 129)',   // green
  'rgb(245, 158, 11)',   // yellow
  'rgb(239, 68, 68)',    // red
  'rgb(139, 92, 246)',   // purple
  'rgb(236, 72, 153)'    // pink
]

const initChart = () => {
  if (!chartCanvas.value || !props.trends.length) return

  if (chartInstance) {
    chartInstance.destroy()
  }

  const ctx = chartCanvas.value.getContext('2d')
  if (!ctx) return

  // 准备标签（日期）
  const labels = props.trends[0]?.data.map(item => {
    const date = new Date(item.date)
    return `${date.getMonth() + 1}/${date.getDate()}`
  }) || []

  // 准备数据集
  const datasets = props.trends.map((trend, index) => ({
    label: trend.accountName,
    data: trend.data.map(item => item.usage),
    borderColor: colors[index % colors.length],
    backgroundColor: 'transparent',
    tension: 0.3
  }))

  chartInstance = new Chart(ctx, {
    type: 'line',
    data: { labels, datasets },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: {
          display: true,
          position: 'top',
          onClick: (_e: unknown, legendItem: { datasetIndex?: number }, legend: { chart: Chart }) => {
            const index = legendItem.datasetIndex!
            const ci = legend.chart
            const meta = ci.getDatasetMeta(index)
            meta.hidden = !meta.hidden
            ci.update()
          }
        },
        tooltip: {
          callbacks: {
            label: (context) => {
              const value = context.parsed?.y ?? 0
              return `${context.dataset.label}: $${value.toFixed(2)}`
            }
          }
        }
      },
      scales: {
        y: {
          beginAtZero: true,
          ticks: {
            callback: (value) => `$${value}`
          }
        }
      }
    }
  })
}

onMounted(() => {
  initChart()
})

watch(() => props.trends, () => {
  initChart()
}, { deep: true })
</script>

<template>
  <div class="w-full h-80">
    <canvas ref="chartCanvas"></canvas>
  </div>
</template>
