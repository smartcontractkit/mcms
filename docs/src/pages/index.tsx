import {useEffect} from 'react';
import {useHistory} from '@docusaurus/router';

export default function Home(): JSX.Element {
  const history = useHistory();

  useEffect(() => {
    // Redirect to /intro immediately
    history.replace('/intro');
  }, [history]);

  // This component will never actually render since we redirect immediately
  return <div>Redirecting...</div>;
}
