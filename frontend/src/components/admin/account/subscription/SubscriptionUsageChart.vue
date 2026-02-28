<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { Chart, registerables } from 'chart.js'
import type { UsageTrend } from '@/types/subscription'

Chart.register(...registerables)

interface Props {
  data: UsageTrend[]  // 近 7 天用量数据
}

const props = defineProps<Props>()

const chartCanvas = ref<HTMLCanvasElement | null>(null)
let chartInstance: Chart | null = null

// 初始化图表
const initChart = () => {
  if (!chartCanvas.value || !props.data.length) return

  // 销毁旧图表
  if (chartInstance) {
    chartInstance.destroy()
  }

  const ctx = chartCanvas.value.getContext('2d')
  if (!ctx) return

  // 准备数据
  const labels = props.data.map(item => {
    const date = new Date(item.date)
    return `${date.getMonth() + 1}/${date.getDate()}`
  })
  const usageData = props.data.map(item => item.usage_usd)
  const limitData = props.data.map(item => item.limit_usd)

  chartInstance = new Chart(ctx, {
    type: 'line',
    data: {
      labels,
      datasets: [
        {
          label: '实际用量',
          data: usageData,
          borderColor: 'rgb(59, 130, 246)',
          backgroundColor: 'rgba(59, 130, 246, 0.1)',
          tension: 0.3,
          fill: true
        },
        {
          label: '限额',
          data: limitData,
          borderColor: 'rgb(239, 68, 68)',
          backgroundColor: 'transparent',
          borderDash: [5, 5],
          tension: 0,
          fill: false
        }
      ]
    },
    options: {
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
              const value = context.parsed.y ?? 0
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

watch(() => props.data, () => {
  initChart()
}, { deep: true })
</script>

<template>
  <div class="w-full h-64">
    <canvas ref="chartCanvas"></canvas>
  </div>
</template>
