import "./page-container.css";

interface Props {
  children: React.ReactNode;
}

export const PageContainer: React.FC<Props> = ({ children }) => {
  return (
    <div className="page-container">
      <div className="header">
        stream navigator
      </div>
      <div>
        {children}
      </div>
    </div>
  );
}

