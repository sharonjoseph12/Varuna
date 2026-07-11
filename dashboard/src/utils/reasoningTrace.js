export function translateReasoning(trace, evidence) {
  if (!trace || !evidence) return "No reasoning trace available.";

  const fragments = [];

  trace.inputs_evaluated.forEach(input => {
    switch(input) {
      case "silence_ratio":
        if (evidence.silence_duration_s) {
          const ratio = (evidence.silence_duration_s / trace.thresholds_used.zone_tolerance_s).toFixed(1);
          fragments.push(`silence exceeded zone tolerance by ${ratio}x (${evidence.silence_duration_s}s)`);
        }
        break;
      case "boundary_proximity":
        if (evidence.boundary_proximity_km !== undefined) {
          fragments.push(`vessel was ${evidence.boundary_proximity_km}km from boundary (within ${trace.thresholds_used.boundary_buffer_km}km buffer)`);
        }
        break;
      case "historical_gap_pattern":
        fragments.push("this gap is anomalous for this vessel's history in this zone");
        break;
      case "time_of_day":
        fragments.push("transit occurred at night");
        break;
      case "kinematic_plausibility":
        if (evidence.conflicting_position) {
           fragments.push("the reported position is physically impossible to reach given the last known position");
        }
        break;
    }
  });

  if (fragments.length === 0) return "Alert fired based on standard zone triggers.";

  return `Alert fired because ${fragments.join(', and ')}. Evaluated by ${trace.engine_version}.`;
}
