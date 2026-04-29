import "./config-list.css";

interface Props {
  configList: string[];
}

export const ConfigList: React.FC<Props> = ({ configList }) => {
  return (
    <div className="config-list">
      {configList.map(item => {
        return (
          <div>
            {item}
          </div>
        )
      })}
    </div>
  );
}

