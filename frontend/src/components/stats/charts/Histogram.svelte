<script>
    /**
     * Histogram — a BarChart with category x-axis.
     * Labels should be the bucket boundary strings (e.g. "0–1", "1–2", …).
     */
    import BarChart from './BarChart.svelte';

    /**
     * @typedef {Object} HistogramProps
     * @property {string[]} labels  - Bucket boundary labels
     * @property {import('chart.js').ChartDataset<'bar'>[]} datasets
     * @property {import('chart.js').ChartOptions<'bar'>} [options]
     * @property {(dataIndex: number, datasetIndex: number) => void} [onBarClick]
     */

    let { labels, datasets, options = {}, onBarClick } = $props();

    const histogramOptions = {
        ...options,
        scales: {
            x: { type: 'category', ...(options?.scales?.x ?? {}) },
            y: { beginAtZero: true, ...(options?.scales?.y ?? {}) }
        },
        plugins: {
            legend: { display: false },
            ...(options?.plugins ?? {})
        }
    };
</script>

<BarChart {labels} {datasets} options={histogramOptions} onBarClick={onBarClick} />
