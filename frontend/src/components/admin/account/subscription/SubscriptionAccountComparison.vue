<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { Chart, registerables } from 'chart.js'

Chart.register(...registerables)

interface AccountComparison {
  accountId: number
  accountName: string
  usage: number
  limit: number
}

interface Props {
  accounts: AccountComparison[]
}

const props = defineProps<Props>()

const chartCanvas = ref<HTMLCanvasElement | null>(null)
let chartInstance: Chart | null = null

const initChart = () => {
  if (!chartCanvas.value || !props.accounts.length) return

  if (chartInstance) {
    chartInstance.destroy()
  }

  const ctx = chartCanvas.value.getContext('2d')
  if (!ctx) return

  const labels = props.accounts.map(acc => acc.accountName)
  const usageData = props.accounts.map(acc => acc.usage)
  const limitData = props.accounts.map(acc => acc.limit)

  chartInstance = new Chart(ctx, {
    type: 'bar',
    data: {
      labels,
      datasets: [
        {
          label: '实际用量',
          data: usageData,
          backgroundColor: 'rgba(59, 130, 246, 0.8)',
          borderColor: 'rgb(59, 130, 246)',
          borderWidth: 1
        },
        {
          label: '限额',
          data: limitData,
          backgroundColor: 'rgba(239, 68, 68, 0.3)',
          borderColor: 'rgb(239, 68, 68)',
          borderWidth: 1
        }
      ]
    },
    options: {
      indexAxis: 'y',
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: {
          display: true,
          position: 'top'
        },
        tooltip: {
          callbacks: {
            label: (context) => {
              const value = context.parsed.x ?? 0
              return `${context.dataset.label}: $${value.toFixed(2)}`
            }
          }
        }
      },
      scales: {
        x: {
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

watch(() => props.accounts, () => {
  initChart()
}, { deep: true })
</script>

<template>
  <div class="w-full h-96">
    <canvas ref="chartCanvas"></canvas>
  </div>
</template>
