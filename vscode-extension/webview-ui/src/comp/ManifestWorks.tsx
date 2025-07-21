import { VSCodeButton, VSCodeDataGrid, VSCodeDataGridCell, VSCodeDataGridRow,  } from '@vscode/webview-ui-toolkit/react';
import { useState, useEffect } from 'react';
import { OcmResource } from '../../../src/data/loader'

export default function ShowManifestWorks() {
    let [manifestWorks, setManifestWorks] = useState<OcmResource[]>([]);
    let [showMore, setShowMore] = useState<Map<any, boolean>>(new Map());

	const updateShowMore = (k: any, v: boolean) => {
        setShowMore(new Map(showMore.set(k, v)));
    }

	useEffect(() => {
        window.addEventListener("message", event => {
			if ('crsDistribution' in event.data && 'ManifestWork' === event.data.crsDistribution.kind) {
				let manifestWorks = JSON.parse(event.data.crsDistribution.crs);
				manifestWorks.forEach((manifestWork: OcmResource) => updateShowMore(manifestWork.name, false));
				setManifestWorks(manifestWorks);
			}
        });
    });

    return (
        <section className="component-row">
            { manifestWorks.length > 0 &&
                <>
                    <h2 style={{ marginTop: '40px' }}>ManifestWorks</h2>
                    <VSCodeDataGrid gridTemplateColumns="1fr 1fr 1fr 1fr 1fr" aria-label='ManifestWorks' >
                        <VSCodeDataGridRow rowType="sticky-header">
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='1'>ManifestWork Name</VSCodeDataGridCell>
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='2'>Namespace</VSCodeDataGridCell>
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='3'>Owner References</VSCodeDataGridCell>
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='4'>Conditions</VSCodeDataGridCell>
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='5'>Resource Status</VSCodeDataGridCell>
                        </VSCodeDataGridRow>

                        {manifestWorks.map(manifestwork => {
                            return <VSCodeDataGridRow>
                                        <VSCodeDataGridCell gridColumn='1'>{manifestwork.kr.metadata.name}</VSCodeDataGridCell>
                                        <VSCodeDataGridCell gridColumn='2'>{manifestwork.kr.metadata.namespace} </VSCodeDataGridCell>
                                        <VSCodeDataGridCell gridColumn='3'>
                                            {manifestwork.kr.metadata.ownerReferences !== undefined && manifestwork.kr.metadata.ownerReferences !== null
                                                ?
                                                    manifestwork.kr.metadata.ownerReferences.map( ( owner:any )=> { return<p> - apiVersion: {owner.apiVersion}, blockOwnerDeletion: {owner.blockOwnerDeletion.toString()}, controller: {owner.controller.toString()}, kind: {owner.kind}, name: {owner.name} </p>  })
                                                :
                                                    ""
                                            }
                                        </VSCodeDataGridCell>
                                        <VSCodeDataGridCell gridColumn='4'>{manifestwork.kr.status.conditions.map( ( condition:any )=> { return<p> - lastTransitionTime: {condition.lastTransitionTime}, message: {condition.message}, observedGeneration: {condition.observedGeneration}, reason: {condition.reason}, status: {condition.status}, type: {condition.type} </p>  })} </VSCodeDataGridCell>
                                        <VSCodeDataGridCell gridColumn='5'>
                                            <VSCodeButton onClick={() => updateShowMore(manifestwork.kr.metadata.name, !showMore.get(manifestwork.kr.metadata.name))}> {showMore.get(manifestwork.kr.metadata.name) ? "Less" : "More"} </VSCodeButton>
                                            {showMore.get(manifestwork.kr.metadata.name)
                                                ?
                                                    manifestwork.kr.status.resourceStatus.manifests.map( ( mf:any )=> { return<p> - group: {mf.resourceMeta.group}, kind: {mf.resourceMeta.kind}, name: {mf.resourceMeta.name}, namespace: {mf.resourceMeta.namespace}, ordinal: {mf.resourceMeta.ordinal}, resource: {mf.resourceMeta.resource}, version: {mf.resourceMeta.version}
                                                        <ul>{mf.conditions.map( ( cond:any )=> { return<li> lastTransitionTime: {cond.lastTransitionTime}, message: {cond.message}, reason: {cond.reason}, status: {cond.status}, type: {cond.type} </li>  })}</ul>
                                                    </p>  })
                                                :
                                                    ''
                                            }
                                        </VSCodeDataGridCell>
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
