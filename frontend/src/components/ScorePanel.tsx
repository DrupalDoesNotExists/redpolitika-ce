"use client";

interface ScorePanelProps {
  cleanliness: number;
  readability: number;
}

function scoreClass(value: number): string {
  if (value >= 7) return "good";
  if (value >= 4) return "mid";
  return "bad";
}

export default function ScorePanel({ cleanliness, readability }: ScorePanelProps) {
  return (
    <div className="score-panel flex items-end gap-10">
      <div className="flex items-center gap-2.5">
        <span className="label">Чистота</span>
        <span className={`score ${scoreClass(cleanliness)}`}>
          {cleanliness.toFixed(1).replace('.', ',')}
          <span className="max"> / 10</span>
        </span>
      </div>

      <div className="flex items-center gap-2.5">
        <span className="label">Читаемость</span>
        <span className={`score ${scoreClass(readability)}`}>
          {readability.toFixed(1).replace('.', ',')}
          <span className="max"> / 10</span>
        </span>
      </div>
    </div>
  );
}
