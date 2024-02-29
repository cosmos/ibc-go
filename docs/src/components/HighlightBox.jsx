import React from "react";

import ReadingIcon from "@site/static/img/icons/hi-reading-icon.svg";
import PrereqIcon from "@site/static/img/icons/hi-prerequisite-icon.svg";
import TargetIcon from "@site/static/img/icons/hi-target-icon.svg";
import StarIcon from "@site/static/img/icons/hi-star-icon.svg";
import TipIcon from "@site/static/img/icons/hi-tip-icon.svg";
import NoteIcon from "@site/static/img/icons/hi-note-icon.svg";
import CoffeeIcon from "@site/static/img/icons/hi-coffee-icon.svg";
import InfoIcon from "@site/static/img/icons/hi-info-icon.svg";
import WarnIcon from "@site/static/img/icons/hi-warn-icon.svg";

const typeToStyles = {
  tip: {
    color1: "#336667",
    color2: "#00B067",
    icon: <TipIcon />,
    darkMode: true,
  },
  reading: {
    color1: "#F46800",
    color2: "#F24CF4",
    icon: <ReadingIcon />,
    darkMode: false,
  },
  info: {
    color1: "#336667",
    color2: "#00B067",
    icon: <InfoIcon />,
    darkMode: true,
  },
  warn: {
    color1: "#00B067",
    color2: "#FFD303",
    icon: <WarnIcon />,
    darkMode: false,
  },
  warning: {
    color1: "#00B067",
    color2: "#FFD303",
    icon: <WarnIcon />,
    darkMode: false,
  },
  synopsis: {
    color1: "#1c1c1c",
    color2: "#1c1c1c",
    icon: null,
    darkMode: true,
  },
  prerequisite: {
    color1: "lightgray",
    color2: "lightgray",
    icon: <PrereqIcon />,
    darkMode: false,
  },
  learning: {
    color1: "#6836D0",
    color2: "#05BDFC",
    icon: <TargetIcon />,
    darkMode: true,
  },
  "best-practice": {
    color1: "#6836D0",
    color2: "#6836D0",
    icon: <StarIcon />,
    darkMode: true,
  },
  remember: {
    color1: "#6D0000",
    color2: "#F66800",
    icon: <TipIcon />,
    darkMode: true,
  },
  note: {
    color1: "#F69900",
    color2: "#FFCE15",
    icon: <NoteIcon />,
    darkMode: false,
  },
  docs: {
    color1: "#6836D0",
    color2: "#F44CF6",
    icon: <CoffeeIcon />,
    darkMode: true,
  },
  // add as many types as you like
};

const gradientStyles = ({ color1, color2 }) => ({
  background: `linear-gradient(78.06deg, ${color1} 1.14%, ${color2} 98.88%)`,
  border: 0,
  borderRadius: 16,
  padding: "0 30px",
  display: "flex",
  width: "100%",
  // alignItems: "center",
  justifyContent: "start",
  marginBottom: 20,
  fontSize: 21,
  flexWrap: "wrap",
  flexDirection: "column",
});

function HighlightBox({ type, title, children }) {
  const styles = typeToStyles[type] || typeToStyles.info; // default to 'info' if type doesn't exist

  const codeStyle = {
    backgroundColor: 'var(--docusaurus-highlighted-code-line-bg)',
  };
  const iconStyles = {
    alignSelf: "center",
    marginTop: "10px",
    filter: styles.darkMode ? "brightness(0) invert(1)" : "brightness(0)",
  };
  const titleStyle = {
    marginTop: "10px",
    marginLeft: "10px",
    color: styles.darkMode ? "#e6e6e6" : "black",
  };
  const childrenStyle = {
    color: styles.darkMode ? "#e6e6e6" : "black",
    marginBottom: "10px",
  };

  const childrenWithStyles = React.Children.map(children, child => {
    if (child.type === 'a') {
      return React.cloneElement(child, {
        style: anchorStyle,
      });
    }
    if (child.type === 'code') {
      return React.cloneElement(child, {
        style: codeStyle,
      });
    }
    return child;
  });

  return (
    <div style={gradientStyles(styles)} className="highlightBox">
      <style>
        {`
          .highlightBox a {
            color: ${styles.darkMode ? "blue" : "#85c1e9"} !important;
          }
        `}
      </style>
      <div style={{ display: "flex", alignItems: "center", marginBottom: "10px" }}>
        <div style={iconStyles}>{styles.icon}</div>
        <h3 style={titleStyle}>{title}</h3>
      </div>
      <div style={childrenStyle}>{childrenWithStyles}</div>
    </div>
  );
}

export default HighlightBox;
