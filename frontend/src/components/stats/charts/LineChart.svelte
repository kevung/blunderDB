<script>
    import { Chart, LineController, LineElement, PointElement, LinearScale, CategoryScale, Tooltip, Legend, Filler } from 'chart.js';
    import { GRIDLINE } from './palette.js';

    Chart.register(LineController, LineElement, PointElement, LinearScale, CategoryScale, Tooltip, Legend, Filler);

    /**
     * @typedef {Object} LineChartProps
     * @property {string[]} labels
     * @property {import('chart.js').ChartDataset<'line'>[]} datasets
     * @property {import('chart.js').ChartOptions<'line'>} [options]
     * @property {import('chart.js').Plugin[]} [plugins]    Per-chart plugins (e.g. grade-band backgrounds).
     * @property {(dataIndex: number, datasetIndex: number, nativeEvent: MouseEvent) => void} [onPointClick]
     */

    let { labels, datasets, options = {}, plugins = [], onPointClick } = $props();

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
            y: { grid: { color: GRIDLINE }, beginAtZero: false }
        }
    };

    $effect(() => {
        if (!canvas) return;

        // Destroy previous instance when reactive inputs change
        if (chart) {
            chart.destroy();
            chart = null;
        }

        const mergedOptions = deepMerge(baseOptions, options ?? {});

        chart = new Chart(canvas, {
            type: 'line',
            data: { labels: labels ?? [], datasets: datasets ?? [] },
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

    /** Shallow-merge two option objects (one level deep). */
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
