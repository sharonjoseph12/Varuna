export const zones = {
  type: "FeatureCollection",
  features: [
    {
      type: "Feature",
      properties: { name: "Coastal Protected Zone", fill_color: "rgba(255, 107, 107, 0.2)", stroke_color: "#FF6B6B" },
      geometry: { type: "Polygon", coordinates: [[[-73.5, 40.5], [-73.0, 40.5], [-73.0, 40.0], [-73.5, 40.0], [-73.5, 40.5]]] }
    },
    {
      type: "Feature",
      properties: { name: "Deep Sea Reserve", fill_color: "rgba(155, 89, 182, 0.2)", stroke_color: "#9B59B6" },
      geometry: { type: "Polygon", coordinates: [[[-72.0, 39.5], [-71.0, 39.5], [-71.0, 38.5], [-72.0, 38.5], [-72.0, 39.5]]] }
    }
  ]
};
