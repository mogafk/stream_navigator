import "./page-container.css";

interface Props {
  children: React.ReactNode;
}

export const PageContainer: React.FC<Props> = ({ children }) => {
  return (
    <div className="page-container">
        <div className="page-content">
            {children}
        </div>
    </div>
  );
}

