import Graph from "graphology";
import { circular } from "graphology-layout";
import forceAtlas2 from "graphology-layout-forceatlas2";
import FA2Layout from "graphology-layout-forceatlas2/worker";
import Sigma from "sigma";
import { PlainObject } from "sigma/types";
import { animateNodes } from "sigma/utils";
import louvain from "graphology-communities-louvain"; // Импорт алгоритма Louvain

// Простая палитра для раскраски кластеров
const palette = [
  "#1f77b4", "#ff7f0e", "#2ca02c", "#d62728", "#9467bd",
  "#8c564b", "#e377c2", "#7f7f7f", "#bcbd22", "#17becf",
];

const run = (data: any) => {
  // Инициализация графа и импорт данных
  const graph = new Graph();
  graph.import(data);

  // === Кластеризация с помощью Louvain ===
  const communities = louvain(graph);
  graph.forEachNode((node) => {
    graph.setNodeAttribute(node, "cluster", communities[node]);
  });

  // === DOM-элементы ===
  const container = document.getElementById("sigma-container") as HTMLElement;
  const FA2Button = document.getElementById("forceatlas2") as HTMLElement;
  const FA2StopLabel = document.getElementById("forceatlas2-stop-label") as HTMLElement;
  const FA2StartLabel = document.getElementById("forceatlas2-start-label") as HTMLElement;
  const randomButton = document.getElementById("random") as HTMLElement;
  const circularButton = document.getElementById("circular") as HTMLElement;

  /** FA2 LAYOUT **/
  // Получаем настройки и запускаем ForceAtlas2 в web worker
  const sensibleSettings = forceAtlas2.inferSettings(graph);
  const fa2Layout = new FA2Layout(graph, {
    settings: sensibleSettings,
  });

  let cancelCurrentAnimation: (() => void) | null = null;

  function stopFA2() {
    fa2Layout.stop();
    FA2StartLabel.style.display = "flex";
    FA2StopLabel.style.display = "none";
  }
  function startFA2() {
    if (cancelCurrentAnimation) cancelCurrentAnimation();
    fa2Layout.start();
    FA2StartLabel.style.display = "none";
    FA2StopLabel.style.display = "flex";
  }
  function toggleFA2Layout() {
    if (fa2Layout.isRunning()) stopFA2();
    else startFA2();
  }
  FA2Button.addEventListener("click", toggleFA2Layout);

  /** RANDOM LAYOUT **/
  function randomLayout() {
    if (fa2Layout.isRunning()) stopFA2();
    if (cancelCurrentAnimation) cancelCurrentAnimation();

    const xExtents = { min: 0, max: 0 };
    const yExtents = { min: 0, max: 0 };
    graph.forEachNode((_node, attributes) => {
      xExtents.min = Math.min(attributes.x, xExtents.min);
      xExtents.max = Math.max(attributes.x, xExtents.max);
      yExtents.min = Math.min(attributes.y, yExtents.min);
      yExtents.max = Math.max(attributes.y, yExtents.max);
    });
    const randomPositions: PlainObject<PlainObject<number>> = {};
    graph.forEachNode((node) => {
      const s = 2;
      randomPositions[node] = {
        x: Math.random() * (xExtents.max * s - xExtents.min * s),
        y: Math.random() * (yExtents.max * s - yExtents.min * s),
      };
    });
    cancelCurrentAnimation = animateNodes(graph, randomPositions, { duration: 2000 });
  }
  randomButton.addEventListener("click", randomLayout);

  /** CIRCULAR LAYOUT **/
  function circularLayout() {
    if (fa2Layout.isRunning()) stopFA2();
    if (cancelCurrentAnimation) cancelCurrentAnimation();

    const circularPositions = circular(graph, { scale: 100 });
    cancelCurrentAnimation = animateNodes(graph, circularPositions, { duration: 2000, easing: "linear" });
  }
  circularButton.addEventListener("click", circularLayout);

  /** Инициализация Sigma с nodeReducer для раскраски по кластерам **/
  const renderer = new Sigma(graph, container, {
    renderEdgeLabels: false,
    defaultNodeColor: "#ccc",
    nodeReducer: (node, data) => ({
      ...data,
      label: node,
      color: palette[data.cluster % palette.length],
    }),
  });

  return () => {
    renderer.kill();
  };
};

document.addEventListener("DOMContentLoaded", () => {
  // Загружаем данные с API и кэшируем в localStorage
  let rawData = localStorage.getItem("graph");
  if (!rawData) {
    fetch("http://localhost:8080/api/graph")
      .then((response) => response.json())
      .then((data) => {
        localStorage.setItem("graph", JSON.stringify(data));
        run(data);
      });
  } else {
    run(JSON.parse(rawData));
  }
});
