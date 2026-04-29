import { PageContainer } from '~/components/PageContainer/page-container'
import "./welcome.css";

export function Welcome() {
  return (
    <PageContainer>
      <main className="flex items-center justify-center pt-16 pb-4">
        <div className="test">
          hello world
        </div>
      </main>
    </PageContainer>

  );
}

