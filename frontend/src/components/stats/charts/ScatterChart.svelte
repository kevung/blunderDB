<script>
    import { Chart, ScatterController, PointElement, LinearScale, Tooltip, Legend } from 'chart.js';
    import { GRIDLINE } from './palette.js';

    Chart.register(ScatterController, PointElement, LinearScale, Tooltip, Legend);

    /**
     * @typedef {Object} ScatterChartProps
     * @property {import('chart.js').ChartDataset<'scatter'>[]} datasets
     * @property {import('chart.js').ChartOptions<'scatter'>} [options]
     * @property {import('chart.js').Plugin[]} [plugins]    Per-chart plugins.
     * @property {(dataIndex: number, datasetIndex: number, nativeEvent: MouseEvent) => void} [onPointClick]
     */

    let { datasets, options = {}, plugins = [], onPointClick } = $props();

    /** @type {HTMLCanvasElement} */
    let canvas;
    /** @type {Chart | null} */
    let chart = null;

    const baseOptions = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
            legend: { display: datasets?.length > 1 },
            tooltip: { mode: 'nearest', intersect: true }
        },
        scales: {
            x: { grid: { color: GRIDLINE } },
            y: { grid: { color: GRIDLINE }, beginAtZero: false }
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
            type: 'scatter',
            data: { datasets: datasets ?? [] },
            plugins: plugins ?? [],
            options: {
                ...mergedOptions,
                onClick: onPointClick
                    ? (evt, elements) => {
                          if (elements.length > 0) {
                              const { datasetIndex, index } = elements[0];
                              onPointClick(index, datasetIndex, evt?.native);
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
