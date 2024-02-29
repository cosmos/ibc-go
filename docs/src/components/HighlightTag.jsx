import React from "react";

const tags = {
  devops: {
    color: "#54ffe0",
    label: "DevOps",
    isBright: true,
  },
  "cosmos-sdk": {
    color: "#F69900",
    label: "Cosmos SDK",
    isBright: true,
  },
  "ibc-go": {
    color: "#ff1717",
    label: "IBC-Go",
  },
  cosmjs: {
    color: "#6836D0",
    label: "CosmJS",
  },
  cosmwasm: {
    color: "#05BDFC",
    label: "CosmWasm",
  },
  cometbft: {
    color: "#00B067",
    label: "CometBFT",
  },
  "cosmos-hub": {
    color: "#f7f199",
    label: "Cosmos Hub",
    isBright: true,
  },
  concepts: {
    color: "#AABAFF",
    label: "Concept",
    isBright: true,
  },
  tutorial: {
    color: "#F46800",
    label: "Tutorial",
  },
  "guided-coding": {
    color: "#F24CF4",
    label: "Guided Coding",
  },
};

const HighlightTag = ({ type, version }) => {
  const styles = tags[type] || tags["ibc-go"]; // default to 'ibc-go' if type doesn't exist

  return (
    <span
      style={{
        backgroundColor: styles.color,
        borderRadius: "2px",
        color: styles.isBright ? "black" : "white",
        padding: "0.3rem",
        marginBottom: "1rem",
        marginRight: "0.25rem",
        display: "inline-block",
      }}
    >
      {styles.label}
      {version ? ` ${version}` : ""}
    </span>
  );
};

export default HighlightTag;
