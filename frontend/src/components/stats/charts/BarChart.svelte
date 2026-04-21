<script>
    import { Chart, BarController, BarElement, LinearScale, CategoryScale, Tooltip, Legend } from 'chart.js';
    import { GRIDLINE } from './palette.js';

    Chart.register(BarController, BarElement, LinearScale, CategoryScale, Tooltip, Legend);

    /**
     * @typedef {Object} BarChartProps
     * @property {string[]} labels
     * @property {import('chart.js').ChartDataset<'bar'>[]} datasets
     * @property {import('chart.js').ChartOptions<'bar'>} [options]
     * @property {(dataIndex: number, datasetIndex: number) => void} [onBarClick]
     */

    let { labels, datasets, options = {}, onBarClick } = $props();

    /** @type {HTMLCanvasElement} */
    let canvas;
    /** @type {Chart | null} */
    let chart = null;

    const baseOptions = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
            legend: { display: datasets?.length > 1 },
            tooltip: { mode: 'index', intersect: false }
        },
        scales: {
            x: { grid: { color: GRIDLINE } },
            y: { grid: { color: GRIDLINE }, beginAtZero: true }
        }
    };

    $effect(() => {
        if (!canvas) return;

        if (chart) {
            chart.destroy();
            chart = null;
        }

        const mergedOptions = deepMerge(baseOptions, options ?? {});

        chart = new Chart(canvas, {
            type: 'bar',
            data: { labels: labels ?? [], datasets: datasets ?? [] },
            options: {
                ...mergedOptions,
                onClick: onBarClick
                    ? (_evt, elements) => {
                          if (elements.length > 0) {
                              const { datasetIndex, index } = elements[0];
                              onBarClick(index, datasetIndex);
                          }
                      }
                    : undefined
            }
        });

        return () => {
            chart?.destroy();
            chart = null;
        };
    });

    function deepMerge(base, override) {
        const result = { ...base };
        for (const key of Object.keys(override)) {
            if (override[key] && typeof override[key] === 'object' && !Array.isArray(override[key]) && base[key]) {
                result[key] = deepMerge(base[key], override[key]);
            } else {
                result[key] = override[key];
            }
        }
        return result;
    }
</script>

<canvas bind:this={canvas}></canvas>
