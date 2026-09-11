import { useLayoutEffect, useMemo, useRef, useState, type PointerEvent } from "react";

import type { StockCandle } from "../types/stock";

const priceFormatter = new Intl.NumberFormat("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 });

function formatTimeline(timestamp: string, intraday: boolean) {
  return new Intl.DateTimeFormat("en-US", intraday
    ? { hour: "2-digit", minute: "2-digit", hour12: false, timeZone: "UTC" }
    : { month: "short", day: "numeric", year: "2-digit", timeZone: "UTC" }).format(new Date(timestamp));
}

export function CandlestickChart({ candles }: { candles: StockCandle[] }) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [pointer, setPointer] = useState<{ x: number; y: number; index: number; time: number } | null>(null);
  const orderedCandles = useMemo(
    () => [...candles].sort((left, right) => new Date(left.timestamp).getTime() - new Date(right.timestamp).getTime()),
    [candles],
  );
  const width = Math.max(760, orderedCandles.length * 12 + 90), height = 330;
  const left = 18, right = 72, top = 18, bottom = 42;
  const minimum = Math.min(...orderedCandles.map((candle) => candle.low));
  const maximum = Math.max(...orderedCandles.map((candle) => candle.high));
  const priceSpan = Math.max(maximum - minimum, 0.01);
  const plotWidth = width - left - right, plotHeight = height - top - bottom;
  const step = plotWidth / Math.max(orderedCandles.length, 1);
  const bodyWidth = Math.max(1.5, Math.min(9, step * 0.58));
  const y = (price: number) => top + ((maximum - price) / priceSpan) * plotHeight;
  const x = (index: number) => left + step * index + step / 2;
  const intraday = new Date(orderedCandles[orderedCandles.length - 1].timestamp).getTime() - new Date(orderedCandles[0].timestamp).getTime() < 2 * 86_400_000;
  const timelineIndexes = Array.from(new Set([0, Math.floor((orderedCandles.length - 1) / 3), Math.floor((orderedCandles.length - 1) * 2 / 3), orderedCandles.length - 1]));

  useLayoutEffect(() => {
    const container = scrollRef.current;
    if (container) container.scrollLeft = container.scrollWidth - container.clientWidth;
  }, [orderedCandles]);

  function selectCandle(event: PointerEvent<SVGSVGElement>) {
    const bounds = event.currentTarget.getBoundingClientRect();
    const rawX = ((event.clientX - bounds.left) / bounds.width) * width;
    const rawY = ((event.clientY - bounds.top) / bounds.height) * height;
    const pointerX = Math.max(left, Math.min(width - right, rawX));
    const pointerY = Math.max(top, Math.min(top + plotHeight, rawY));
    const decimalIndex = Math.max(0, Math.min(orderedCandles.length - 1, (pointerX - left - step / 2) / step));
    const lowerIndex = Math.floor(decimalIndex);
    const upperIndex = Math.min(orderedCandles.length - 1, Math.ceil(decimalIndex));
    const fraction = decimalIndex - lowerIndex;
    const lowerTime = new Date(orderedCandles[lowerIndex].timestamp).getTime();
    const upperTime = new Date(orderedCandles[upperIndex].timestamp).getTime();
    setPointer({ x: pointerX, y: pointerY, index: Math.round(decimalIndex), time: lowerTime + (upperTime - lowerTime) * fraction });
  }

  const hovered = pointer === null ? null : orderedCandles[pointer.index];
  const pointerPrice = pointer === null ? 0 : maximum - ((pointer.y - top) / plotHeight) * priceSpan;
  const tooltipWidth = 176;
  const tooltipX = pointer === null ? left : Math.min(Math.max(pointer.x + 10, left), width - right - tooltipWidth);
  const timeBadgeWidth = 88;
  const timeBadgeX = pointer === null ? left : Math.max(left, Math.min(width - right - timeBadgeWidth, pointer.x - timeBadgeWidth / 2));

  return <div className="candle-chart-scroll" ref={scrollRef} aria-label="Scrollable historical price chart">
  <svg className="candle-chart" width={width} viewBox={`0 0 ${width} ${height}`} role="img" aria-label="Interactive historical candlestick price chart"
    onPointerMove={selectCandle} onPointerDown={selectCandle} onPointerLeave={() => setPointer(null)}>
    {[0, 1, 2, 3, 4].map((line) => {
      const lineY = top + (plotHeight / 4) * line;
      const price = maximum - (priceSpan / 4) * line;
      return <g key={line}><line x1={left} x2={width-right} y1={lineY} y2={lineY} className="chart-grid" />
        <text x={width-right+8} y={lineY+4} className="chart-axis-label">${priceFormatter.format(price)}</text></g>;
    })}
    {orderedCandles.map((candle, index) => {
      const candleX = x(index), rising = candle.close >= candle.open;
      const candleTop = y(Math.max(candle.open, candle.close));
      const bodyHeight = Math.max(1.5, Math.abs(y(candle.open) - y(candle.close)));
      return <g key={candle.timestamp} className={rising ? "candle-up" : "candle-down"}><line x1={candleX} x2={candleX} y1={y(candle.high)} y2={y(candle.low)} /><rect x={candleX-bodyWidth/2} y={candleTop} width={bodyWidth} height={bodyHeight} rx="1" /></g>;
    })}
    {timelineIndexes.map((index) => <text key={index} x={x(index)} y={height-12} textAnchor={index === 0 ? "start" : index === orderedCandles.length-1 ? "end" : "middle"} className="chart-axis-label">{formatTimeline(orderedCandles[index].timestamp, intraday)}</text>)}
    {hovered && pointer && <g className="chart-hover">
      <line x1={pointer.x} x2={pointer.x} y1={top} y2={top+plotHeight} className="chart-crosshair" />
      <line x1={left} x2={width-right} y1={pointer.y} y2={pointer.y} className="chart-crosshair" />
      <circle cx={pointer.x} cy={pointer.y} r="3" />
      <rect x={width-right} y={pointer.y-10} width={right} height="20" rx="4" className="chart-axis-badge" />
      <text x={width-right+5} y={pointer.y+4} className="chart-axis-badge-text">${priceFormatter.format(pointerPrice)}</text>
      <rect x={timeBadgeX} y={height-bottom+7} width={timeBadgeWidth} height="20" rx="4" className="chart-axis-badge" />
      <text x={timeBadgeX+timeBadgeWidth/2} y={height-bottom+21} textAnchor="middle" className="chart-axis-badge-text">{formatTimeline(new Date(pointer.time).toISOString(), intraday)}</text>
      <rect x={tooltipX} y={top+8} width={tooltipWidth} height="104" rx="8" className="chart-tooltip" />
      <text x={tooltipX+10} y={top+27} className="chart-tooltip-title">{formatTimeline(hovered.timestamp, intraday)} UTC</text>
      <text x={tooltipX+10} y={top+47}>O {priceFormatter.format(hovered.open)}   H {priceFormatter.format(hovered.high)}</text>
      <text x={tooltipX+10} y={top+67}>L {priceFormatter.format(hovered.low)}   C {priceFormatter.format(hovered.close)}</text>
      <text x={tooltipX+10} y={top+88}>Volume {hovered.volume.toLocaleString("en-US")}</text>
    </g>}
    <rect x={left} y={top} width={plotWidth} height={plotHeight} className="chart-hit-area" />
  </svg>
  </div>;
}
