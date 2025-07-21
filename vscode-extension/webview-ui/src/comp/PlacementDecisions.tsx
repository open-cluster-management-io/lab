import { VSCodeDataGrid, VSCodeDataGridCell, VSCodeDataGridRow,  } from '@vscode/webview-ui-toolkit/react';
import { useState, useEffect } from 'react';
import { OcmResource } from '../../../src/data/loader'

export default function ShowPlacementDecisions() {
    let [placementDecisions, setPlacementDecisions] = useState<OcmResource[]>([]);

	useEffect(() => {
        window.addEventListener("message", event => {
			if ('crsDistribution' in event.data && 'PlacementDecision' === event.data.crsDistribution.kind) {
				setPlacementDecisions(JSON.parse(event.data.crsDistribution.crs));
			}
		});
    });

    return (
        <section className="component-row">
            { placementDecisions.length > 0 &&
                <>
                    <h2 style={{ marginTop: '40px' }}>Placement Decisions</h2>
                    <VSCodeDataGrid gridTemplateColumns="1fr 1fr 1fr" aria-label='PlacementDecisions' >
                        <VSCodeDataGridRow rowType="sticky-header">
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='1'>Placement Decision Name</VSCodeDataGridCell>
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='2'>Namespace</VSCodeDataGridCell>
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='3'>Decisions</VSCodeDataGridCell>
                        </VSCodeDataGridRow>

                        {placementDecisions.map(placementDecision => {
                            return <VSCodeDataGridRow>
                                        <VSCodeDataGridCell gridColumn='1'>{placementDecision.kr.metadata.name}</VSCodeDataGridCell>
                                        <VSCodeDataGridCell gridColumn='2'>{placementDecision.kr.metadata.namespace} </VSCodeDataGridCell>
                                        <VSCodeDataGridCell gridColumn='3'>{placementDecision.kr.status.decisions.map( ( dec:any )=> { return<p> - clusterName: {dec.clusterName}, reason: {dec.reason} </p>  })} </VSCodeDataGridCell>
                                    </VSCodeDataGridRow>
                        } )
                        }
                    </VSCodeDataGrid>
                    <div style={{ borderTop: "1px solid #fff ", marginLeft: 10, marginRight: 10 }}></div>
                </>
            }
        </section>
    );
}
