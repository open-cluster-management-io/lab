import { useState, useEffect } from 'react';
import { ConnectedContext } from '../../../src/data/builder'

export default function ShowSelectedContext() {
    let [selectedContext, setSelectedContext] = useState<ConnectedContext>();

	useEffect(() => {
        window.addEventListener("message", event => {
			if ('selectedContext' in event.data) {
				setSelectedContext(JSON.parse(event.data.selectedContext));
			}
		});
    });

	return (
        <section>
			{ selectedContext &&
				<>
					<h1>{ selectedContext.name }</h1>
					<p><b>Cluster:</b> <br/>{ selectedContext.cluster.name }</p>
					<p><b>User:</b> <br/>{ selectedContext.user.name }</p>
				</>
			}
		</section>
    );
}
